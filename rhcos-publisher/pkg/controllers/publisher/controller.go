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

// Package publisher implements the marketplace publisher: for a staged VHD it
// configures the corresponding marketplace plans and image version as a draft
// through the Partner Center Product Ingestion API. It never submits the
// draft for publishing — a human reviews and publishes in Partner Center.
package publisher

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-HCP/internal/utils"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/reconciler"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/storage"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

// ControllerName is the publisher's controller name, used for the workqueue
// name, logging and metrics.
const ControllerName = "marketplace-publisher"

// InstallerCache is the installer lister's read interface.
type InstallerCache interface {
	Get(key string) (stream.Image, bool)
}

// StorageClient resolves staged blob state and URLs.
type StorageClient interface {
	BlobExists(ctx context.Context, blobPath string) (bool, error)
	WebURL(ctx context.Context, blobPath string) (string, error)
}

// StatusClient reads and updates RHCOSRelease resources.
type StatusClient interface {
	Get(ctx context.Context, branch string) (*v1alpha1.RHCOSRelease, error)
	UpdateArchStatus(ctx context.Context, branch, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error
}

// Syncer configures marketplace drafts for staged VHDs.
type Syncer struct {
	ingestionClient marketplace.IngestionClient
	installerCache  InstallerCache
	storageClient   StorageClient
	statusClient    StatusClient
	cfg             *config.Config
}

// NewSyncer builds the publisher syncer.
func NewSyncer(
	ingestionClient marketplace.IngestionClient,
	installerCache InstallerCache,
	storageClient StorageClient,
	statusClient StatusClient,
	cfg *config.Config,
) *Syncer {
	return &Syncer{
		ingestionClient: ingestionClient,
		installerCache:  installerCache,
		storageClient:   storageClient,
		statusClient:    statusClient,
		cfg:             cfg,
	}
}

// SyncOnce configures the marketplace draft of one branch/architecture pair.
func (s *Syncer) SyncOnce(ctx context.Context, key string) error {
	logger := utils.LoggerFromContext(ctx)

	branchName, arch, err := reconciler.ParseKey(key)
	if err != nil {
		logger.Error(err, "dropping malformed key")
		return nil
	}
	branch, configured := s.cfg.Branch(branchName)
	if !configured {
		logger.Info("branch no longer configured; ignoring", "branch", branchName)
		return nil
	}
	image, known := s.installerCache.Get(key)
	if !known {
		logger.Info("no installer image cached; ignoring", "key", key)
		return nil
	}

	minorID, err := branch.MinorID()
	if err != nil {
		return err
	}
	version, err := marketplace.ImageVersion(minorID, image.Release)
	if err != nil {
		return err
	}
	skus, err := marketplace.SKUsForArch(branch, arch)
	if err != nil {
		return err
	}

	release, err := s.statusClient.Get(ctx, branchName)
	if err != nil {
		return err
	}
	currentStatus := release.Status.Architectures[arch]
	if currentStatus.Phase == v1alpha1.ImagePhaseDraft && currentStatus.Release == image.Release {
		logger.Info("draft already configured", "release", image.Release, "version", version)
		return nil
	}

	// The draft references the staged VHD by URL; it must be there.
	blobPath := storage.BlobPath(branchName, arch, image.Release)
	staged, err := s.storageClient.BlobExists(ctx, blobPath)
	if err != nil {
		return err
	}
	if !staged {
		return fmt.Errorf("VHD %s is not staged yet", blobPath)
	}
	stagedURL, err := s.storageClient.WebURL(ctx, blobPath)
	if err != nil {
		return err
	}

	jobID, err := s.configureDraft(ctx, skus, version, stagedURL)
	if err != nil {
		return err
	}

	logger.Info("marketplace draft configured", "release", image.Release, "version", version, "jobID", jobID)
	return s.statusClient.UpdateArchStatus(ctx, branchName, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
		archStatus.Release = image.Release
		archStatus.Phase = v1alpha1.ImagePhaseDraft
		archStatus.StagedURL = stagedURL
		archStatus.ConfigureJobID = jobID
	})
}

// configureDraft ensures each SKU's plan exists and carries the new image
// version, submitting all modified technical configurations in one configure
// job. It returns the job ID of the final configure call ("" when nothing
// needed changing).
func (s *Syncer) configureDraft(ctx context.Context, skus []marketplace.SKU, version, stagedURL string) (string, error) {
	logger := utils.LoggerFromContext(ctx)

	productID, err := s.ingestionClient.GetProductIDByExternalID(ctx, s.cfg.Marketplace.Publisher, s.cfg.Marketplace.Offer)
	if err != nil {
		return "", err
	}
	tree, err := s.ingestionClient.GetResourceTree(ctx, productID)
	if err != nil {
		return "", err
	}

	// Create any missing plans first: their durable IDs are needed to build
	// the technical configurations.
	var newPlans []map[string]any
	for _, sku := range skus {
		if _, found := marketplace.FindPlanDurableID(tree, sku.Name); !found {
			logger.Info("creating marketplace plan", "plan", sku.Name)
			newPlans = append(newPlans, marketplace.NewPlanResource(productID, sku.Name, sku.Name))
		}
	}
	if len(newPlans) > 0 {
		jobID, err := s.ingestionClient.Configure(ctx, newPlans)
		if err != nil {
			return "", err
		}
		if err := s.ingestionClient.WaitForJob(ctx, jobID); err != nil {
			return "", err
		}
		tree, err = s.ingestionClient.GetResourceTree(ctx, productID)
		if err != nil {
			return "", err
		}
	}

	var changedConfigs []map[string]any
	for _, sku := range skus {
		planID, found := marketplace.FindPlanDurableID(tree, sku.Name)
		if !found {
			return "", fmt.Errorf("plan %s not found after creation", sku.Name)
		}
		techConfig, found := marketplace.FindTechConfig(tree, planID)
		if !found {
			techConfig = marketplace.NewTechConfigResource(productID, planID)
		}
		changed := marketplace.EnsureSKUs(techConfig, []marketplace.SKU{sku})
		if marketplace.EnsureImageVersion(techConfig, version, []marketplace.VMImage{{ImageType: sku.ImageType, URI: stagedURL}}) {
			changed = true
		}
		if changed {
			changedConfigs = append(changedConfigs, techConfig)
		}
	}
	if len(changedConfigs) == 0 {
		return "", nil
	}

	jobID, err := s.ingestionClient.Configure(ctx, changedConfigs)
	if err != nil {
		return "", err
	}
	if err := s.ingestionClient.WaitForJob(ctx, jobID); err != nil {
		return "", err
	}
	return jobID, nil
}
