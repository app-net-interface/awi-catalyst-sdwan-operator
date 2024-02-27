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

package connection_status

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv1 "app-net-interface.io/kube-awi/api/awi/v1alpha1"
	awiClient "app-net-interface.io/kube-awi/client"
	awi "github.com/app-net-interface/awi-grpc/pb"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func WatchStatusUpdates(ctx context.Context,
	awiClient *awiClient.AwiGrpcClient,
	k8sClient k8sclient.Client,
	interval time.Duration) {
	logger := ctrl.Log.WithName("status-update-watcher")
	checkConnectionsStatuses(awiClient, logger, k8sClient)
	checkAppConnectionsStatuses(awiClient, logger, k8sClient)
	// TODO make configurable
	ticker := time.NewTicker(interval)
	for {
		select {
		case t := <-ticker.C:
			logger.Info("Periodic status check", "time", t)
			checkConnectionsStatuses(awiClient, logger, k8sClient)
			checkAppConnectionsStatuses(awiClient, logger, k8sClient)
		case <-ctx.Done():
			return
		}
	}
}

func checkConnectionsStatuses(awiClient *awiClient.AwiGrpcClient, logger logr.Logger,
	k8sClient k8sclient.Client) {
	connections, err := awiClient.ListConnections()
	if err != nil {
		logger.Error(err, "failed to list connections in awi grpc server")
		return
	}
	connectionsMap := make(map[string]*awi.ConnectionInformation, len(connections))
	for _, conn := range connections {
		connectionsMap[conn.GetId()] = conn
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var interNetworkDomainList apiv1.InterNetworkDomainList
	err = k8sClient.List(ctx, &interNetworkDomainList)
	if err != nil {
		logger.Error(err, "failed to list InterNetworkDomain CRDs")
		return
	}

	for _, crd := range interNetworkDomainList.Items {
		conn, ok := connectionsMap[awiClient.GetConnectionId(&crd.Spec)]
		if !ok {
			logger.Error(err, "couldn't find connection matching to CRD",
				"namespace", crd.GetNamespace(), "name", crd.GetName())
			continue
		}
		logger.Info("Checking status of InterNetworkDomain item",
			"namespace", crd.GetNamespace(), "name", crd.GetName(),
			"CRD current status", crd.Status, "connection status", conn.GetStatus(),
			"connection string status", awi.Status_name[int32(conn.GetStatus())])
		if crd.Status.State == awi.Status_name[int32(conn.GetStatus())] &&
			crd.Status.ConnectionId == awiClient.GetConnectionId(&crd.Spec) {
			continue
		}
		crd.Status.State = awi.Status_name[int32(conn.GetStatus())]
		crd.Status.ConnectionId = awiClient.GetConnectionId(&crd.Spec)
		err = k8sClient.Status().Update(ctx, &crd)
		if err != nil {
			logger.Error(err, "couldn't update InterNetworkDomain CRD status",
				"namespace", crd.GetNamespace(), "name", crd.GetName(),
				"status", crd.Status)
			continue
		}
	}
}

func checkAppConnectionsStatuses(awiClient *awiClient.AwiGrpcClient, logger logr.Logger,
	k8sClient k8sclient.Client) {
	appConnections, err := awiClient.ListAppConnections()
	if err != nil {
		logger.Error(err, "failed to list appConnections in awi grpc server")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var appConnectionList apiv1.InterNetworkDomainAppConnectionList
	err = k8sClient.List(ctx, &appConnectionList)
	if err != nil {
		logger.Error(err, "failed to list InterNetworkDomainAppConnection CRDs")
		return
	}

	for _, crd := range appConnectionList.Items {
		for _, appConn := range appConnections {
			// looking for appConnection matching to CRD
			if !(crd.Spec.AppConnection.GetNetworkDomainConnection().GetSelector().GetMatchName() == appConn.GetAppConnectionConfig().GetNetworkDomainConnection().GetSelector().GetMatchName()) ||
				!(crd.Spec.AppConnection.GetMetadata().GetName() == appConn.GetAppConnectionConfig().GetMetadata().GetName()) {
				continue
			}
			logger.Info("Checking status of InterNetworkDomainAppConnection item",
				"namespace", crd.GetNamespace(), "name", crd.GetName(),
				"CRD current status", crd.Status, "connection status", appConn.GetStatus(),
				"connection string status", awi.Status_name[int32(appConn.GetStatus())])
			if crd.Status == awi.Status_name[int32(appConn.GetStatus())] {
				break
			}
			crd.Status = awi.Status_name[int32(appConn.GetStatus())]
			err = k8sClient.Status().Update(ctx, &crd)
			if err != nil {
				logger.Error(err, "couldn't update InterNetworkDomainAppConnection CRD status",
					"namespace", crd.GetNamespace(), "name", crd.GetName(),
					"status", crd.Status)
				break
			}
			break
		}
	}
}
