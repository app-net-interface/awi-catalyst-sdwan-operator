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

//+kubebuilder:rbac:groups=awi.cisco.awi,resources=vpcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=vpcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=vpcs/finalizers,verbs=update

type VPCSyncer struct {
	k8sClient k8s_cl.Client
	awiClient *awi_cl.AwiGrpcClient
	logger    logr.Logger
}

func (s *VPCSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var err error
	var existingVPCs []*awi.VPC
	for _, cloud := range SupportedClouds {
		cloudVPCs, err := s.awiClient.ListVPCs(cloud)
		existingVPCs = append(existingVPCs, cloudVPCs...)
		if err != nil {
			return err
		}
	}

	var vpcList apiv1.VPCList
	err = s.k8sClient.List(ctx, &vpcList, k8s_cl.InNamespace(Namespace))
	if err != nil {
		return err
	}
	vpcCRDMap := make(map[string]apiv1.VPC, len(vpcList.Items))
	for _, vpc := range vpcList.Items {
		vpcCRDMap[vpc.GetName()] = vpc
	}

	for _, vpc := range existingVPCs {
		_, ok := vpcCRDMap[getVpcCRDName(vpc)]
		if ok {
			// if it's already present remove it from map
			delete(vpcCRDMap, getVpcCRDName(vpc))
			continue
		}
		newVPCCRD := apiv1.VPC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getVpcCRDName(vpc),
				Namespace: Namespace,
			},
			Spec: *vpc,
		}
		s.logger.Info("Adding new VPC CRD", "name", newVPCCRD.GetName())
		err := s.k8sClient.Create(ctx, &newVPCCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest
	for _, vpcCRD := range vpcCRDMap {
		s.logger.Info("Removing VPC CRD", "name", vpcCRD.GetName())
		err := s.k8sClient.Delete(ctx, &vpcCRD)
		if err != nil {
			return err
		}
	}
	return nil
}

func getVpcCRDName(vpc *awi.VPC) string {
	return fmt.Sprintf("%s.%s",
		strings.ToLower(vpc.GetProvider()), vpc.GetID())
}
