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
	k8s_cl "sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "awi.cisco.awi/api/v1"
	awi_cl "awi.cisco.awi/client"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

//+kubebuilder:rbac:groups=awi.cisco.awi,resources=subnets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=subnets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=subnets/finalizers,verbs=update

type SubnetSyncer struct {
	k8sClient k8s_cl.Client
	awiClient *awi_cl.AwiGrpcClient
	logger    logr.Logger
}

type subnetWithProvider struct {
	Subnet   *awi.Subnet
	Provider string
}

func (s *SubnetSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var err error

	var existingSubnets []subnetWithProvider
	for _, cloud := range SupportedClouds {
		cloudSubnets, err := s.awiClient.ListSubnets(cloud)
		if err != nil {
			return err
		}
		for _, subnet := range cloudSubnets {
			withProvider := subnetWithProvider{
				Subnet:   subnet,
				Provider: cloud,
			}
			existingSubnets = append(existingSubnets, withProvider)
		}
	}

	var subnetList apiv1.SubnetList
	err = s.k8sClient.List(ctx, &subnetList, k8s_cl.InNamespace(Namespace))
	if err != nil {
		return err
	}
	subnetCRDMap := make(map[string]apiv1.Subnet, len(subnetList.Items))
	for _, subnet := range subnetList.Items {
		subnetCRDMap[subnet.GetName()] = subnet
	}

	for _, subnet := range existingSubnets {
		_, ok := subnetCRDMap[getSubnetCRDName(subnet)]
		if ok {
			// if it's already present remove it from map
			delete(subnetCRDMap, getSubnetCRDName(subnet))
			continue
		}
		newSubnetCRD := apiv1.Subnet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getSubnetCRDName(subnet),
				Namespace: Namespace,
			},
			Spec: *subnet.Subnet,
		}
		s.logger.Info("Adding new Subnet CRD", "name", newSubnetCRD.GetName())
		err := s.k8sClient.Create(ctx, &newSubnetCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest
	for _, subnetCRD := range subnetCRDMap {
		s.logger.Info("Removing Subnet CRD", "name", subnetCRD.GetName())
		err := s.k8sClient.Delete(ctx, &subnetCRD)
		if err != nil {
			return err
		}
	}
	return nil
}

func getSubnetCRDName(subnet subnetWithProvider) string {
	return fmt.Sprintf("%s.%s", strings.ToLower(subnet.Provider), subnet.Subnet.GetSubnetId())
}
