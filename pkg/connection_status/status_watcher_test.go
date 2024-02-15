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

	awiv1 "awi.cisco.awi/api/v1"
	"awi.cisco.awi/client"
	awi "github.com/app-net-interface/awi-grpc/pb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	awiMock "github.com/app-net-interface/awi-grpc/mocks"
)

const (
	name      string = "conn"
	namespace string = "default"
	// Context parameters
	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

var _ = Describe("Status watcher", func() {
	Context("Status updates", func() {
		It("should update connection status based on response from awi grpc server", func() {
			By("By creating a new Connection")
			// Create a new Connection object
			connLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			ctx := context.Background()
			connSvc := &awiv1.InterNetworkDomain{
				TypeMeta:   metav1.TypeMeta{APIVersion: "awi.cisco.awi/v1", Kind: "InterNetworkDomain"},
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: awi.ConnectionRequest{
					Metadata: &awi.ConnectionMetadata{},
					Spec: &awi.NetworkDomainConnectionConfig{
						Source: &awi.NetworkDomainConnectionConfig_Source{
							NetworkDomain: &awi.NetworkDomainConnectionConfig_NetworkDomain{
								Selector: &awi.NetworkDomainConnectionConfig_Selector{
									MatchName: &awi.NetworkDomainConnectionConfig_MatchName{
										Name: "AWS VPC development",
									},
									MatchId: &awi.NetworkDomainConnectionConfig_MatchId{
										Id: "vpc-111",
									},
								},
							},
						},
						Destination: &awi.NetworkDomainConnectionConfig_Destination{
							NetworkDomain: &awi.NetworkDomainConnectionConfig_NetworkDomain{
								Selector: &awi.NetworkDomainConnectionConfig_Selector{
									MatchName: &awi.NetworkDomainConnectionConfig_MatchName{
										Name: "VPN 10",
									},
									MatchId: &awi.NetworkDomainConnectionConfig_MatchId{
										Id: "10",
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, connSvc)).Should(Succeed())

			By("connection status returned from awi server should be registered in CRD")
			mockConnectionController := awiMock.NewConnectionControllerClient(GinkgoT())
			mockConnectionController.On("ListConnections",
				mock.Anything, mock.Anything).Return(&awi.ListConnectionsResponse{
				Connections: []*awi.ConnectionInformation{
					{
						Id: "vpc-111:10",
						Metadata: &awi.ConnectionMetadata{
							Name: "example-conn",
						},
						Status: awi.Status_SUCCESS,
					},
				},
			}, nil)
			mockAppConnectionController := awiMock.NewAppConnectionControllerClient(GinkgoT())
			mockAppConnectionController.On("ListConnectedApps",
				mock.Anything, mock.Anything).Return(&awi.ListAppConnectionsResponse{
				AppConnections: nil,
			}, nil)

			awiClient := &client.AwiGrpcClient{
				ConnectionControllerClient:    mockConnectionController,
				AppConnectionControllerClient: mockAppConnectionController,
			}
			ctxWithCancel, cancel := context.WithCancel(ctx)
			defer cancel()
			go WatchStatusUpdates(ctxWithCancel, awiClient, k8sClient, time.Millisecond*100)

			Eventually(func() bool {
				connObj := &awiv1.InterNetworkDomain{}
				err := k8sClient.Get(ctx, connLookupKey, connObj)
				if err == nil && connObj.Status.ConnectionId == "vpc-111:10" &&
					connObj.Status.State == "SUCCESS" {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

			By("connection status updated in awi server should be updated in CRD")
			mockConnectionController = awiMock.NewConnectionControllerClient(GinkgoT())
			mockConnectionController.On("ListConnections",
				mock.Anything, mock.Anything).Return(&awi.ListConnectionsResponse{
				Connections: []*awi.ConnectionInformation{
					{
						Id: "vpc-111:10",
						Metadata: &awi.ConnectionMetadata{
							Name: "example-conn",
						},
						Status: awi.Status_FAILED,
					},
				},
			}, nil)
			awiClient.ConnectionControllerClient = mockConnectionController
			Eventually(func() bool {
				connObj := &awiv1.InterNetworkDomain{}
				err := k8sClient.Get(ctx, connLookupKey, connObj)
				if err == nil && connObj.Status.ConnectionId == "vpc-111:10" &&
					connObj.Status.State == "FAILED" {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should update app connection status based on response from awi grpc server", func() {
			By("By creating a new InterNetworkDomainAppConnection")
			// Create a new Connection object

			appConnName := "db_to_db"
			clusterCconnectionId := "vpc-111:10"
			connLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			ctx := context.Background()
			appConn := &awiv1.InterNetworkDomainAppConnection{
				TypeMeta:   metav1.TypeMeta{APIVersion: "awi.cisco.awi/v1", Kind: "InterNetworkDomainAppConnection"},
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: awiv1.AppConnectionSpec{

					AppConnection: awi.AppConnection{
						Metadata: &awi.AppMetadata{
							Name: appConnName,
						},
						NetworkDomainConnection: &awi.NetworkDomainConnection{
							Selector: &awi.NetworkDomainConnection_Selector{MatchName: clusterCconnectionId},
						},
						To: &awi.To{
							Endpoint: &awi.Endpoint{
								Selector: &awi.Endpoint_Selector{
									MatchLabels: map[string]string{
										"app": "database",
										"env": "staging",
									},
								},
							},
						},
						From: &awi.From{
							Endpoint: &awi.Endpoint{
								Selector: &awi.Endpoint_Selector{
									MatchLabels: map[string]string{
										"app": "database",
										"env": "development",
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, appConn)).Should(Succeed())

			By("connection status returned from awi server should be registered in CRD")
			mockConnectionController := awiMock.NewConnectionControllerClient(GinkgoT())
			mockConnectionController.On("ListConnections",
				mock.Anything, mock.Anything).Return(&awi.ListConnectionsResponse{
				Connections: []*awi.ConnectionInformation{
					{
						Id: clusterCconnectionId,
						Metadata: &awi.ConnectionMetadata{
							Name: "example-conn",
						},
						Status: awi.Status_SUCCESS,
					},
				},
			}, nil)
			mockAppConnectionController := awiMock.NewAppConnectionControllerClient(GinkgoT())
			mockAppConnectionController.On("ListConnectedApps",
				mock.Anything, mock.Anything).Return(&awi.ListAppConnectionsResponse{
				AppConnections: []*awi.AppConnectionInformation{
					{
						Id: "this_is_random",
						AppConnectionConfig: &awi.AppConnection{
							Controller: "",
							Metadata: &awi.AppMetadata{
								Name: appConnName,
							},
							NetworkDomainConnection: &awi.NetworkDomainConnection{
								Selector: &awi.NetworkDomainConnection_Selector{
									MatchName: clusterCconnectionId,
								},
							},
						},
						NetworkDomainConnectionName: clusterCconnectionId,
						Status:                      awi.Status_SUCCESS,
					},
					{
						Id: "other_id",
						AppConnectionConfig: &awi.AppConnection{
							Controller: "",
							Metadata: &awi.AppMetadata{
								Name: "other_app_conn",
							},
							NetworkDomainConnection: &awi.NetworkDomainConnection{
								Selector: &awi.NetworkDomainConnection_Selector{
									MatchName: clusterCconnectionId,
								},
							},
						},
						NetworkDomainConnectionName: clusterCconnectionId,

						Status: awi.Status_IN_PROGRESS,
					},
				},
			}, nil)

			awiClient := &client.AwiGrpcClient{
				ConnectionControllerClient:    mockConnectionController,
				AppConnectionControllerClient: mockAppConnectionController,
			}
			ctxWithCancel, cancel := context.WithCancel(ctx)
			defer cancel()
			go WatchStatusUpdates(ctxWithCancel, awiClient, k8sClient, time.Millisecond*100)

			Eventually(func() bool {
				connObj := &awiv1.InterNetworkDomainAppConnection{}
				err := k8sClient.Get(ctx, connLookupKey, connObj)
				if err == nil && connObj.Status == "SUCCESS" {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

			By("app connection status updated in awi server should be updated in CRD")
			mockAppConnectionController = awiMock.NewAppConnectionControllerClient(GinkgoT())
			mockAppConnectionController.On("ListConnectedApps",
				mock.Anything, mock.Anything).Return(&awi.ListAppConnectionsResponse{
				AppConnections: []*awi.AppConnectionInformation{
					{
						Id: "this_is_random",
						AppConnectionConfig: &awi.AppConnection{
							Controller: "",
							Metadata: &awi.AppMetadata{
								Name: appConnName,
							},
							NetworkDomainConnection: &awi.NetworkDomainConnection{
								Selector: &awi.NetworkDomainConnection_Selector{
									MatchName: clusterCconnectionId,
								},
							},
						},
						Status: awi.Status_FAILED,
					},
					{
						Id: "other_id",
						AppConnectionConfig: &awi.AppConnection{
							Controller: "",
							Metadata: &awi.AppMetadata{
								Name: "other_app_conn",
							},
							NetworkDomainConnection: &awi.NetworkDomainConnection{
								Selector: &awi.NetworkDomainConnection_Selector{
									MatchName: clusterCconnectionId,
								},
							},
						},
						Status: awi.Status_IN_PROGRESS,
					},
				},
			}, nil)
			awiClient.AppConnectionControllerClient = mockAppConnectionController
			Eventually(func() bool {
				connObj := &awiv1.InterNetworkDomainAppConnection{}
				err := k8sClient.Get(ctx, connLookupKey, connObj)
				if err == nil && connObj.Status == "FAILED" {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})
})
