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

	apiv1 "app-net-interface.io/kube-awi/api/awi/v1alpha1"
	awi_cl "app-net-interface.io/kube-awi/client"
	awi "github.com/app-net-interface/awi-grpc/pb"
)

type InstanceSyncer struct {
	k8sClient k8s_cl.Client
	awiClient *awi_cl.AwiGrpcClient
	logger    logr.Logger
}

type instanceWithProvider struct {
	Instance *awi.Instance
	Provider string
}

//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=instances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=instances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=instances/finalizers,verbs=update

func (s *InstanceSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var err error

	var existingInstances []instanceWithProvider
	for _, cloud := range SupportedClouds {
		cloudInstances, err := s.awiClient.ListInstances(cloud)
		if err != nil {
			return err
		}
		for _, instance := range cloudInstances {
			withProvider := instanceWithProvider{
				Instance: instance,
				Provider: cloud,
			}
			existingInstances = append(existingInstances, withProvider)
		}
	}

	var instanceList apiv1.InstanceList
	err = s.k8sClient.List(ctx, &instanceList, k8s_cl.InNamespace(Namespace))
	if err != nil {
		return err
	}
	instanceCRDMap := make(map[string]apiv1.Instance, len(instanceList.Items))
	for _, instance := range instanceList.Items {
		instanceCRDMap[instance.GetName()] = instance
	}

	for _, instance := range existingInstances {
		_, ok := instanceCRDMap[getInstanceCRDName(instance)]
		if ok {
			// if it's already present remove it from map
			delete(instanceCRDMap, getInstanceCRDName(instance))
			continue
		}
		newInstanceCRD := apiv1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getInstanceCRDName(instance),
				Namespace: Namespace,
			},
			Spec: *instance.Instance,
		}
		s.logger.Info("Adding new Instance CRD", "name", newInstanceCRD.GetName())
		err := s.k8sClient.Create(ctx, &newInstanceCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest
	for _, instanceCRD := range instanceCRDMap {
		s.logger.Info("Removing Instance CRD", "name", instanceCRD.GetName())
		err := s.k8sClient.Delete(ctx, &instanceCRD)
		if err != nil {
			return err
		}
	}
	return nil
}

func getInstanceCRDName(instance instanceWithProvider) string {
	return fmt.Sprintf("%s.%s", strings.ToLower(instance.Provider), instance.Instance.GetID())
}
