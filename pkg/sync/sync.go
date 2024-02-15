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
	awi_cl "awi.cisco.awi/client"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	k8s_cl "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// TODO make configurable
var SupportedClouds = []string{"AWS"}

const Namespace = "awi-system"

type Syncer interface {
	Sync() error
}

type Syncers struct {
	allSyncers []Syncer
	logger     logr.Logger
}

func NewSyncers(k8sClient k8s_cl.Client, awiClient *awi_cl.AwiGrpcClient) *Syncers {
	logger := ctrl.Log.WithName("sync-logger")
	syncers := &Syncers{
		logger: logger,
	}
	syncers.allSyncers = []Syncer{
		&InstanceSyncer{
			k8sClient: k8sClient,
			awiClient: awiClient,
			logger:    logger,
		},
		&SiteSyncer{
			k8sClient: k8sClient,
			awiClient: awiClient,
			logger:    logger,
		},
		&SubnetSyncer{
			k8sClient: k8sClient,
			awiClient: awiClient,
			logger:    logger,
		},
		&VPCSyncer{
			k8sClient: k8sClient,
			awiClient: awiClient,
			logger:    logger,
		},
		&VPNSyncer{
			k8sClient: k8sClient,
			awiClient: awiClient,
			logger:    logger,
		},
		&NetworkDomainSyncer{
			k8sClient: k8sClient,
			logger:    logger,
		},
	}
	return syncers
}

func (s *Syncers) Sync() {
	s.logger.Info("Starting to sync objects...")
	for _, syncer := range s.allSyncers {
		s.logger.Info("Syncing", "syncer", fmt.Sprintf("%T", syncer))
		err := syncer.Sync()
		if err != nil {
			s.logger.Error(err, fmt.Sprintf("Failure during sync of %T", syncer))
		}
	}
}

func (s *Syncers) StartPeriodicSync(ctx context.Context) {
	s.Sync()
	// TODO make time configurable
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case t := <-ticker.C:
			s.logger.Info("Periodic objects sync", "time", t)
			s.Sync()
		case <-ctx.Done():
			return
		}
	}
}
