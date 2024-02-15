// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http:www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	ctrl "sigs.k8s.io/controller-runtime"

	awi "github.com/app-net-interface/awi-grpc/pb"
)

type AwiGrpcClient struct {
	logger                        logr.Logger
	grpcConn                      *grpc.ClientConn
	ConnectionControllerClient    awi.ConnectionControllerClient
	AppConnectionControllerClient awi.AppConnectionControllerClient
	CloudClient                   awi.CloudClient
}

func NewClient(awiCatalystAddress string) *AwiGrpcClient {
	awiClient := &AwiGrpcClient{}
	awiClient.WithLogger()
	awiClient.WithConnection(awiCatalystAddress)
	awiClient.WithGrpcClients()
	return awiClient
}

func (awiClient *AwiGrpcClient) WithLogger() {
	awiClient.logger = ctrl.Log.WithName("grpc-client")
}

func (awiClient *AwiGrpcClient) WithConnection(awiCatalystAddress string) {
	awiClient.logger.Info("connecting to grpc server", "address", awiCatalystAddress)
	var err error
	awiClient.grpcConn, err = grpc.Dial(awiCatalystAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect to grpc server at %s", awiCatalystAddress)
	}
	awiClient.logger.Info("connected to grpc server at %s", "address", awiCatalystAddress)
}

func (awiClient *AwiGrpcClient) WithGrpcClients() {
	awiClient.ConnectionControllerClient = awi.NewConnectionControllerClient(awiClient.grpcConn)
	awiClient.AppConnectionControllerClient = awi.NewAppConnectionControllerClient(awiClient.grpcConn)
	awiClient.CloudClient = awi.NewCloudClient(awiClient.grpcConn)
}

func (awiClient *AwiGrpcClient) ConnectionRequest(connSpec *awi.ConnectionRequest) error {
	if connSpec == nil {
		return fmt.Errorf("empty connection spec")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	awiClient.logger.Info("sending connection request", "connection name", connSpec.GetMetadata().GetName())
	response, err := awiClient.ConnectionControllerClient.Connect(ctx, connSpec)
	if err != nil {
		return fmt.Errorf("error recevived from connection request: %v", err)
	}
	awiClient.logger.Info("connection response", "response", response)
	return nil
}

func (awiClient *AwiGrpcClient) DisconnectRequest(connSpec *awi.ConnectionRequest) error {
	if connSpec == nil {
		return fmt.Errorf("empty connection spec")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	awiClient.logger.Info("sending disconnect request", "connection name", connSpec.GetMetadata().GetName())
	connId := awiClient.GetConnectionId(connSpec)
	response, err := awiClient.ConnectionControllerClient.Disconnect(ctx, &awi.DisconnectRequest{
		ConnectionId: connId,
	})
	if err != nil {
		return fmt.Errorf("error recevived from disconnection request: %v", err)
	}
	awiClient.logger.Info("disconnect response", "response", response)
	return nil
}

func (awiClient *AwiGrpcClient) AppConnectionRequest(connSpec *awi.AppConnection) error {
	if connSpec == nil {
		return fmt.Errorf("empty app connection spec")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	awiClient.logger.Info("sending app connection request", "app connection name", connSpec.GetMetadata().GetName())
	response, err := awiClient.AppConnectionControllerClient.ConnectApps(ctx, connSpec)
	if err != nil {
		return fmt.Errorf("error recevived from app connection request: %v", err)
	}
	awiClient.logger.Info("app connection response", "response", response)
	return nil
}

func (awiClient *AwiGrpcClient) AppDisconnectRequest(connSpec *awi.AppConnection) error {
	if connSpec == nil {
		return fmt.Errorf("empty app connection spec")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connections, err := awiClient.AppConnectionControllerClient.ListConnectedApps(ctx, &awi.ListAppConnectionsRequest{})
	if err != nil {
		return err
	}

	// we don't know ID of app connection, so we look for matching one
	id := ""
	for _, conn := range connections.GetAppConnections() {
		if conn.GetAppConnectionConfig().GetNetworkDomainConnection().GetSelector().GetMatchName() == connSpec.GetNetworkDomainConnection().GetSelector().GetMatchName() &&
			conn.GetAppConnectionConfig().GetMetadata().GetName() == connSpec.GetMetadata().GetName() {
			id = conn.GetId()
			break
		}
	}
	if id == "" {
		awiClient.logger.Info("Couldn't find app connection matching to this connection spec",
			"AppConnSpec", connSpec)
		return nil
	}

	awiClient.logger.Info("sending app disconnect request", "id", id)
	response, err := awiClient.AppConnectionControllerClient.DisconnectApps(ctx, &awi.AppDisconnectionRequest{
		ConnectionId: id,
	})
	if err != nil {
		return fmt.Errorf("error recevived from app disconnection request: %v", err)
	}
	awiClient.logger.Info("app disconnect response", "response", response)
	return nil
}

func (awiClient *AwiGrpcClient) ListConnections() ([]*awi.ConnectionInformation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connections, err := awiClient.ConnectionControllerClient.ListConnections(ctx, &awi.ListConnectionsRequest{})
	if err != nil {
		awiClient.logger.Error(err, "failed to list connections")
		return nil, err
	}
	return connections.GetConnections(), nil
}

func (awiClient *AwiGrpcClient) ListAppConnections() ([]*awi.AppConnectionInformation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connections, err := awiClient.AppConnectionControllerClient.ListConnectedApps(ctx, &awi.ListAppConnectionsRequest{})
	if err != nil {
		awiClient.logger.Error(err, "failed to list app connections")
		return nil, err
	}
	return connections.GetAppConnections(), nil
}

func (awiClient *AwiGrpcClient) GetConnectionId(connSpec *awi.ConnectionRequest) string {
	return fmt.Sprintf("%s:%s",
		connSpec.GetSpec().GetSource().GetNetworkDomain().GetSelector().GetMatchId().GetId(),
		connSpec.GetSpec().GetDestination().GetNetworkDomain().GetSelector().GetMatchId().GetId())
}

func (awiClient *AwiGrpcClient) ListVPCs(provider string) ([]*awi.VPC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	vpcsResp, err := awiClient.CloudClient.ListVPCs(ctx, &awi.ListVPCRequest{
		Provider: provider,
	})
	if err != nil {
		return nil, err
	}
	return vpcsResp.VPCs, nil
}

func (awiClient *AwiGrpcClient) ListInstances(provider string) ([]*awi.Instance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	instancesResp, err := awiClient.CloudClient.ListInstances(ctx, &awi.ListInstancesRequest{
		Provider: provider,
	})
	if err != nil {
		return nil, err
	}
	return instancesResp.Instances, nil
}

func (awiClient *AwiGrpcClient) ListSites() ([]*awi.SiteDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	siteResp, err := awiClient.CloudClient.ListSites(ctx, &awi.ListSiteRequest{})
	if err != nil {
		return nil, err
	}
	return siteResp.Sites, nil
}

func (awiClient *AwiGrpcClient) ListSubnets(provider string) ([]*awi.Subnet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	subnetResp, err := awiClient.CloudClient.ListSubnets(ctx, &awi.ListSubnetRequest{
		Provider: provider,
	})
	if err != nil {
		return nil, err
	}
	return subnetResp.Subnets, nil
}

func (awiClient *AwiGrpcClient) ListVPNs() ([]*awi.VPN, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	vpnResp, err := awiClient.CloudClient.ListVPNs(ctx, &awi.ListVPNRequest{})
	if err != nil {
		return nil, err
	}
	return vpnResp.VPNs, nil
}
