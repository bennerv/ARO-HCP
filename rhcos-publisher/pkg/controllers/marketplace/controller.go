// Copyright 2026 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package marketplace implements the marketplace lister: it polls the Azure
// Marketplace for the image versions that exist in the ARO offering's SKUs,
// caches them, and enqueues a reconcile key whenever a branch/architecture
// pair's published versions change.
package marketplace

import (
	"context"
	"errors"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/Azure/ARO-HCP/internal/utils"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
)

// ControllerName is the marketplace lister's controller name, used for the
// workqueue name, logging and metrics.
const ControllerName = "marketplace-lister"

// SyncKey is the single key the periodic ticker enqueues.
const SyncKey = "sync"

// StatusClient is the subset of the status client used by the lister.
type StatusClient interface {
	UpdateStatus(ctx context.Context, branch string, mutate func(*v1alpha1.RHCOSReleaseStatus)) error
}

// Syncer polls the marketplace image versions of all configured SKUs.
type Syncer struct {
	versionLister    marketplace.VersionLister
	statusClient     StatusClient
	cfg              *config.Config
	enqueueReconcile func(ctx context.Context, key string)

	mu       sync.RWMutex
	synced   bool
	versions map[string]sets.Set[string] // key: SKU name
}

// NewSyncer builds the marketplace lister syncer. enqueueReconcile feeds keys
// into the reconciler controller's queue.
func NewSyncer(versionLister marketplace.VersionLister, statusClient StatusClient, cfg *config.Config, enqueueReconcile func(ctx context.Context, key string)) *Syncer {
	return &Syncer{
		versionLister:    versionLister,
		statusClient:     statusClient,
		cfg:              cfg,
		enqueueReconcile: enqueueReconcile,
		versions:         map[string]sets.Set[string]{},
	}
}

// HasVersion reports whether the marketplace serves the given image version
// under the SKU.
func (s *Syncer) HasVersion(sku, version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[sku].Has(version)
}

// Synced reports whether at least one full sync succeeded for every
// configured SKU. The reconciler refuses to act before that.
func (s *Syncer) Synced() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.synced
}

// SyncOnce lists the marketplace versions of every configured branch's SKUs,
// updates the cache, and enqueues reconcile keys for changed
// branch/architecture pairs.
func (s *Syncer) SyncOnce(ctx context.Context, _ string) error {
	logger := utils.LoggerFromContext(ctx)

	var errs []error
	allSucceeded := true
	for _, branch := range s.cfg.Branches {
		branchSucceeded := true
		for _, arch := range []string{config.ArchX86_64, config.ArchAarch64} {
			skus, err := marketplace.SKUsForArch(branch, arch)
			if err != nil {
				allSucceeded = false
				branchSucceeded = false
				errs = append(errs, fmt.Errorf("branch %s/%s: %w", branch.Name, arch, err))
				continue
			}

			changed := false
			for _, sku := range skus {
				versionList, err := s.versionLister.ListVersions(ctx, s.cfg.Marketplace.Publisher, s.cfg.Marketplace.Offer, sku.Name)
				if err != nil {
					allSucceeded = false
					branchSucceeded = false
					errs = append(errs, fmt.Errorf("sku %s: %w", sku.Name, err))
					continue
				}
				if s.updateSKUCache(sku.Name, versionList) {
					changed = true
				}
			}
			if changed {
				key := branch.Name + "/" + arch
				logger.Info("marketplace versions changed", "key", key)
				s.enqueueReconcile(ctx, key)
			}
		}

		if branchSucceeded {
			now := metav1.Now()
			if err := s.statusClient.UpdateStatus(ctx, branch.Name, func(releaseStatus *v1alpha1.RHCOSReleaseStatus) {
				releaseStatus.LastMarketplaceSync = &now
			}); err != nil {
				errs = append(errs, fmt.Errorf("branch %s: %w", branch.Name, err))
			}
		}
	}

	if allSucceeded {
		s.mu.Lock()
		s.synced = true
		s.mu.Unlock()
	}
	return errors.Join(errs...)
}

// updateSKUCache replaces one SKU's cached version set and reports whether it
// differed.
func (s *Syncer) updateSKUCache(sku string, versionList []string) bool {
	fresh := sets.New(versionList...)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, seen := s.versions[sku]
	s.versions[sku] = fresh
	return !seen || !previous.Equal(fresh)
}
