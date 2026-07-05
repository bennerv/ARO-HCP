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

// Package installer implements the installer lister: it polls the
// openshift/installer repository for the coreos stream metadata of each
// configured branch, caches the pinned Azure images, and enqueues a reconcile
// key whenever a branch/architecture pair's image changes.
package installer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-HCP/internal/utils"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

// ControllerName is the installer lister's controller name, used for the
// workqueue name, logging and metrics.
const ControllerName = "installer-lister"

// SyncKey is the single key the periodic ticker enqueues.
const SyncKey = "sync"

// StatusClient is the subset of the status client used by the lister.
type StatusClient interface {
	UpdateStatus(ctx context.Context, branch string, mutate func(*v1alpha1.RHCOSReleaseStatus)) error
}

// StreamClient fetches the Azure images pinned by a branch's coreos stream.
type StreamClient interface {
	FetchImages(ctx context.Context, branch string, rhelVersion int) ([]stream.Image, error)
}

// Syncer polls the coreos stream metadata of all configured branches.
type Syncer struct {
	streamClient     StreamClient
	statusClient     StatusClient
	cfg              *config.Config
	enqueueReconcile func(ctx context.Context, key string)

	mu     sync.RWMutex
	synced bool
	images map[string]stream.Image // key: "{branch}/{arch}"
}

// NewSyncer builds the installer lister syncer. enqueueReconcile feeds keys
// into the reconciler controller's queue.
func NewSyncer(streamClient StreamClient, statusClient StatusClient, cfg *config.Config, enqueueReconcile func(ctx context.Context, key string)) *Syncer {
	return &Syncer{
		streamClient:     streamClient,
		statusClient:     statusClient,
		cfg:              cfg,
		enqueueReconcile: enqueueReconcile,
		images:           map[string]stream.Image{},
	}
}

// Get returns the cached image of a branch/architecture key.
func (s *Syncer) Get(key string) (stream.Image, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	image, ok := s.images[key]
	return image, ok
}

// Synced reports whether at least one full sync succeeded for every
// configured branch. The reconciler refuses to act before that.
func (s *Syncer) Synced() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.synced
}

// SyncOnce fetches the stream metadata of every configured branch, updates
// the cache, and enqueues reconcile keys for changed branch/architecture
// pairs. A branch whose fetch fails keeps its cached state; the error is
// returned so the workqueue retries with backoff.
func (s *Syncer) SyncOnce(ctx context.Context, _ string) error {
	logger := utils.LoggerFromContext(ctx)

	var errs []error
	allSucceeded := true
	for _, branch := range s.cfg.Branches {
		images, err := s.streamClient.FetchImages(ctx, branch.Name, branch.RHELVersion)
		if err != nil {
			allSucceeded = false
			errs = append(errs, fmt.Errorf("branch %s: %w", branch.Name, err))
			continue
		}

		for _, changedKey := range s.updateBranchCache(branch.Name, images) {
			logger.Info("installer image changed", "key", changedKey)
			s.enqueueReconcile(ctx, changedKey)
		}

		now := metav1.Now()
		if err := s.statusClient.UpdateStatus(ctx, branch.Name, func(releaseStatus *v1alpha1.RHCOSReleaseStatus) {
			releaseStatus.LastInstallerSync = &now
		}); err != nil {
			errs = append(errs, fmt.Errorf("branch %s: %w", branch.Name, err))
		}
	}

	if allSucceeded {
		s.mu.Lock()
		s.synced = true
		s.mu.Unlock()
	}
	return errors.Join(errs...)
}

// updateBranchCache replaces the cached images of one branch and returns the
// keys whose image was added, changed or removed.
func (s *Syncer) updateBranchCache(branchName string, images []stream.Image) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	fresh := map[string]stream.Image{}
	for _, image := range images {
		fresh[image.Key()] = image
	}

	var changed []string
	prefix := branchName + "/"
	for key, oldImage := range s.images {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		newImage, stillThere := fresh[key]
		if !stillThere {
			delete(s.images, key)
			changed = append(changed, key)
			continue
		}
		if newImage != oldImage {
			s.images[key] = newImage
			changed = append(changed, key)
		}
		delete(fresh, key)
	}
	for key, image := range fresh {
		s.images[key] = image
		changed = append(changed, key)
	}
	return changed
}
