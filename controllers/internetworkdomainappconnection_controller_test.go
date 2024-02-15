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

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:docs-gen:collapse=Apache License

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awiv1 "awi.cisco.awi/api/v1"
	awiMock "github.com/app-net-interface/awi-grpc/mocks"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

var _ = Describe("InterNetworkDomainAppConnection Controller", func() {
	const (
		//Meta
		appConnectionName   = "sample-appconnection-svc"
		namespace           = "default"
		clusterConnectionId = "vpc-111:10"
		connectionID        = "baby_shark"
	)
	It("send should send connection request to grpc server whenever new CRD is created", func() {
		appConnectionRequestSpec := &awi.AppConnection{
			Metadata: &awi.AppMetadata{
				Name: appConnectionName,
			},

			NetworkDomainConnection: &awi.NetworkDomainConnection{
				Selector: &awi.NetworkDomainConnection_Selector{MatchName: clusterConnectionId},
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
		}

		By("creating object connection request should be sent")
		t := GinkgoT()
		mockConnectionController := awiMock.NewAppConnectionControllerClient(t)
		defer mockConnectionController.AssertExpectations(t)

		// using context to make sure expected function was called and wait for this call
		// to happen
		creaCtx, creCancel := context.WithCancel(context.Background())
		awiTestClient.AppConnectionControllerClient = mockConnectionController
		mockConnectionController.EXPECT().
			ConnectApps(mock.Anything, mock.Anything).
			Run(func(context.Context, *awi.AppConnection, ...grpc.CallOption) {
				creCancel()
			}).
			Return(&awi.AppConnectionResponse{}, nil)

		appConn := &awiv1.InterNetworkDomainAppConnection{
			TypeMeta:   metav1.TypeMeta{APIVersion: "awi.cisco.awi/v1", Kind: "InterNetworkDomainAppConnection"},
			ObjectMeta: metav1.ObjectMeta{Name: appConnectionName, Namespace: namespace},
			Spec: awiv1.AppConnectionSpec{
				AppConnection: *appConnectionRequestSpec,
			},
		}

		Expect(k8sClient.Create(ctx, appConn)).Should(Succeed())
		select {
		case _ = <-creaCtx.Done():
		case <-time.After(5 * time.Second):
			t.Errorf("Deadline for create call to mock connection controller exceeded")
		}

		By("removing object disconnect should be sent")
		mockConnectionController.On("ListConnectedApps",
			mock.Anything, mock.Anything).Return(&awi.ListAppConnectionsResponse{
			AppConnections: []*awi.AppConnectionInformation{
				{
					Id: connectionID,
					AppConnectionConfig: &awi.AppConnection{
						Controller: "",
						Metadata: &awi.AppMetadata{
							Name: appConnectionName,
						},
						NetworkDomainConnection: &awi.NetworkDomainConnection{
							Selector: &awi.NetworkDomainConnection_Selector{
								MatchName: clusterConnectionId,
							},
						},
					},
					Status: awi.Status_SUCCESS,
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
								MatchName: clusterConnectionId,
							},
						},
					},
					Status: awi.Status_IN_PROGRESS,
				},
			},
		}, nil)
		delCtx, delCanc := context.WithCancel(context.Background())
		mockConnectionController.EXPECT().
			DisconnectApps(mock.Anything, mock.Anything).
			Run(func(_ context.Context, req *awi.AppDisconnectionRequest, _ ...grpc.CallOption) {
				Expect(req.GetConnectionId()).To(Equal(connectionID))
				delCanc()
			}).
			Return(&awi.AppDisconnectionResponse{}, nil)
		Expect(k8sClient.Delete(ctx, appConn)).Should(Succeed())
		select {
		case _ = <-delCtx.Done():
		case <-time.After(5 * time.Second):
			t.Errorf("Deadline for delete call to mock connection controller exceeded")
		}
	})
})
