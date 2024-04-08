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

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_cl "sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "app-net-interface.io/kube-awi/api/awi/v1alpha1"
	awi_cl "app-net-interface.io/kube-awi/client"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=vpns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=vpns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=vpns/finalizers,verbs=update

type VPNSyncer struct {
	k8sClient k8s_cl.Client
	awiClient *awi_cl.AwiGrpcClient
	logger    logr.Logger
}

func (s *VPNSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	existingVPNs, err := s.awiClient.ListVPNs()
	if err != nil {
		return err
	}

	var vpnList apiv1.VPNList
	err = s.k8sClient.List(ctx, &vpnList, k8s_cl.InNamespace(Namespace))
	if err != nil {
		return err
	}
	vpnCRDMap := make(map[string]apiv1.VPN, len(vpnList.Items))
	for _, vpn := range vpnList.Items {
		vpnCRDMap[vpn.GetName()] = vpn
	}

	for _, vpn := range existingVPNs {
		_, ok := vpnCRDMap[getVPNCRDName(vpn)]
		if ok {
			// if it's already present remove it from map
			delete(vpnCRDMap, getVPNCRDName(vpn))
			continue
		}
		newVPNCRD := apiv1.VPN{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getVPNCRDName(vpn),
				Namespace: Namespace,
			},
			Spec: *vpn,
		}
		s.logger.Info("Adding new VPN CRD", "name", newVPNCRD.GetName())
		err := s.k8sClient.Create(ctx, &newVPNCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest
	for _, vpnCRD := range vpnCRDMap {
		s.logger.Info("Removing VPN CRD", "name", vpnCRD.GetName())
		err := s.k8sClient.Delete(ctx, &vpnCRD)
		if err != nil {
			return err
		}
	}
	return nil
}

func getVPNCRDName(vpn *awi.VPN) string {
	return fmt.Sprintf("%s", vpn.GetID())
}
