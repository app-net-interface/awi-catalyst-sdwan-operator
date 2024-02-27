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

//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=sites,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=sites/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.app-net-interface.io,resources=sites/finalizers,verbs=update

type SiteSyncer struct {
	k8sClient k8s_cl.Client
	awiClient *awi_cl.AwiGrpcClient
	logger    logr.Logger
}

func (s *SiteSyncer) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	existingSites, err := s.awiClient.ListSites()
	if err != nil {
		return err
	}

	var siteList apiv1.SiteList
	err = s.k8sClient.List(ctx, &siteList, k8s_cl.InNamespace(Namespace))
	if err != nil {
		return err
	}
	siteCRDMap := make(map[string]apiv1.Site, len(siteList.Items))
	for _, site := range siteList.Items {
		siteCRDMap[site.GetName()] = site
	}

	for _, site := range existingSites {
		_, ok := siteCRDMap[getSiteCRDName(site)]
		if ok {
			// if it's already present remove it from map
			delete(siteCRDMap, getSiteCRDName(site))
			continue
		}
		newSiteCRD := apiv1.Site{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getSiteCRDName(site),
				Namespace: Namespace,
			},
			Spec: *site,
		}
		s.logger.Info("Adding new Site CRD", "name", newSiteCRD.GetName())
		err := s.k8sClient.Create(ctx, &newSiteCRD)
		if err != nil {
			return err
		}
	}

	// all still existing were removed from map, we delete the rest
	for _, siteCRD := range siteCRDMap {
		s.logger.Info("Removing Site CRD", "name", siteCRD.GetName())
		err := s.k8sClient.Delete(ctx, &siteCRD)
		if err != nil {
			return err
		}
	}
	return nil
}

func getSiteCRDName(site *awi.SiteDetail) string {
	return fmt.Sprintf("%s", strings.ToLower(site.GetID()))
}
