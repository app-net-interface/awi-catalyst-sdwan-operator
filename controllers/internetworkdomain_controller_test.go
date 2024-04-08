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

	awiv1alpha1 "app-net-interface.io/kube-awi/api/awi/v1alpha1"
	awiMock "github.com/app-net-interface/awi-grpc/mocks"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

var _ = Describe("InternNetworkDomain Controller", func() {
	const (
		//Meta
		internetworkdomainName = "sample-connection-svc"
		namespace              = "default"
	)

	It("send should send connection request to grpc server whenever new CRD is created", func() {
		connectionRequestSpec := &awi.ConnectionRequest{
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
		}

		t := GinkgoT()
		mockConnectionController := awiMock.NewConnectionControllerClient(t)
		defer mockConnectionController.AssertExpectations(t)
		awiTestClient.ConnectionControllerClient = mockConnectionController
		// using context to make sure expected function was called and wait for this call
		// to happen
		creaCtx, creCancel := context.WithCancel(context.Background())
		mockConnectionController.EXPECT().
			Connect(mock.Anything, connectionRequestSpec).
			Run(func(_ context.Context, _ *awi.ConnectionRequest, _ ...grpc.CallOption) {
				creCancel()
			}).
			Return(&awi.ConnectionResponse{}, nil)

		connSvc := &awiv1alpha1.InterNetworkDomain{
			TypeMeta:   metav1.TypeMeta{APIVersion: "awi.app-net-interface.io/v1alpha1", Kind: "InterNetworkDomain"},
			ObjectMeta: metav1.ObjectMeta{Name: internetworkdomainName, Namespace: namespace},
			Spec:       *connectionRequestSpec,
		}
		Expect(k8sClient.Create(ctx, connSvc)).Should(Succeed())
		select {
		case _ = <-creaCtx.Done():
		case <-time.After(5 * time.Second):
			t.Errorf("Deadline for create call to mock connection controller exceeded")
		}

		By("removing object")
		delCtx, delCanc := context.WithCancel(context.Background())
		mockConnectionController.EXPECT().
			Disconnect(mock.Anything, mock.Anything).
			Run(func(_ context.Context, _ *awi.DisconnectRequest, _ ...grpc.CallOption) {
				delCanc()
			}).
			Return(&awi.DisconnectResponse{}, nil)

		Expect(k8sClient.Delete(ctx, connSvc)).Should(Succeed())
		select {
		case _ = <-delCtx.Done():
		case <-time.After(5 * time.Second):
			t.Errorf("Deadline for delete call to mock connection controller exceeded")
		}
	})
})
