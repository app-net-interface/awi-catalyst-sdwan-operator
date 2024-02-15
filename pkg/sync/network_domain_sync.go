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
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "awi.cisco.awi/api/v1"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

//+kubebuilder:rbac:groups=awi.cisco.awi,resources=networkdomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=networkdomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=networkdomains/finalizers,verbs=update

type NetworkDomainSyncer struct {
	logger    logr.Logger
	k8sClient k8sclient.Client
}

// Sync creates NetworkDomains which are based on existing VPCs and VPNs CRDs
func (s *NetworkDomainSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var vpcList apiv1.VPCList
	err := s.k8sClient.List(ctx, &vpcList, k8sclient.InNamespace(Namespace))
	if err != nil {
		return err
	}

	var vpnList apiv1.VPNList
	err = s.k8sClient.List(ctx, &vpnList, k8sclient.InNamespace(Namespace))
	if err != nil {
		return err
	}

	var existingNetworkDomainList apiv1.NetworkDomainList
	err = s.k8sClient.List(ctx, &existingNetworkDomainList, k8sclient.InNamespace(Namespace))
	if err != nil {
		return err
	}
	ndCRDMap := make(map[string]apiv1.NetworkDomain, len(existingNetworkDomainList.Items))
	for _, nd := range existingNetworkDomainList.Items {
		ndCRDMap[nd.GetName()] = nd
	}

	for _, vpc := range vpcList.Items {
		crdName := getVPCNetworkDomainCRDName(&vpc)
		_, ok := ndCRDMap[crdName]
		if ok {
			// if it's already present remove it from map
			delete(ndCRDMap, crdName)
			continue
		}
		newNDCRD := apiv1.NetworkDomain{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crdName,
				Namespace: Namespace,
				Labels: map[string]string{
					"discovered": "yes",
				},
			},
			Spec: awi.NetworkDomainObject{
				Type:      "VPC",
				Name:      vpc.Spec.GetName(),
				Id:        vpc.Spec.GetID(),
				Provider:  strings.ToUpper(vpc.Spec.GetProvider()),
				AccountId: "",  //TODO
				Labels:    nil, //TODO
			},
		}
		s.logger.Info("Adding new NetworkDomain CRD", "name", newNDCRD.GetName())
		err := s.k8sClient.Create(ctx, &newNDCRD)
		if err != nil {
			return err
		}
	}

	for _, vpn := range vpnList.Items {
		crdName := getVPNNetworkDomainCRDName(&vpn)
		_, ok := ndCRDMap[crdName]
		if ok {
			// if it's already present remove it from map
			delete(ndCRDMap, crdName)
			continue
		}
		newNDCRD := apiv1.NetworkDomain{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crdName,
				Namespace: Namespace,
				Labels: map[string]string{
					"discovered": "yes",
				},
			},
			Spec: awi.NetworkDomainObject{
				Type:      "VRF",
				Name:      vpn.Spec.GetID(),
				Id:        vpn.Spec.GetID(),
				AccountId: "",  //TODO
				Labels:    nil, //TODO
			},
		}
		s.logger.Info("Adding new NetworkDomain CRD", "name", newNDCRD.GetName())
		err := s.k8sClient.Create(ctx, &newNDCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest,
	// but only those discovered
	for _, ndCRD := range ndCRDMap {
		if ndCRD.GetLabels() != nil && ndCRD.GetLabels()["discovered"] != "yes" {
			continue
		}
		s.logger.Info("Removing NetworkDomain CRD", "name", ndCRD.GetName())
		err := s.k8sClient.Delete(ctx, &ndCRD)
		if err != nil {
			return err
		}
	}

	return nil
}

func getVPCNetworkDomainCRDName(vpc *apiv1.VPC) string {
	return fmt.Sprintf("vpc.%s.%s.%s", strings.ToLower(vpc.Spec.GetProvider()), vpc.Spec.GetName(), vpc.Spec.GetID())
}

func getVPNNetworkDomainCRDName(vpn *apiv1.VPN) string {
	return fmt.Sprintf("vpn.%s", strings.ToLower(vpn.Spec.GetID()))
}
