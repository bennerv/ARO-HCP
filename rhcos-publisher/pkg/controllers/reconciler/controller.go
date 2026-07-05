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

// Package reconciler implements the controller that compares the installer
// lister's view (which RHCOS images exist) with the marketplace lister's view
// (which are published) and stages or purges VHDs accordingly.
package reconciler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/ARO-HCP/internal/utils"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/storage"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

// ControllerName is the reconciler's controller name, used for the workqueue
// name, logging and metrics.
const ControllerName = "image-reconciler"

// InstallerCache is the installer lister's read interface.
type InstallerCache interface {
	Get(key string) (stream.Image, bool)
	Synced() bool
}

// MarketplaceCache is the marketplace lister's read interface.
type MarketplaceCache interface {
	HasVersion(sku, version string) bool
	Synced() bool
}

// StorageClient stages and purges VHD blobs.
type StorageClient interface {
	BlobExists(ctx context.Context, blobPath string) (bool, error)
	UploadFile(ctx context.Context, blobPath, filePath string) (string, error)
	DeleteBlob(ctx context.Context, blobPath string) error
	ListBlobs(ctx context.Context, prefix string) ([]string, error)
	WebURL(ctx context.Context, blobPath string) (string, error)
}

// Downloader fetches and verifies a VHD locally.
type Downloader interface {
	FetchVHD(ctx context.Context, image stream.Image) (string, error)
}

// StatusClient reads and updates RHCOSRelease resources.
type StatusClient interface {
	Get(ctx context.Context, branch string) (*v1alpha1.RHCOSRelease, error)
	UpdateArchStatus(ctx context.Context, branch, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error
}

// Syncer reconciles one branch/architecture pair at a time.
type Syncer struct {
	installerCache   InstallerCache
	marketplaceCache MarketplaceCache
	storageClient    StorageClient
	downloader       Downloader
	statusClient     StatusClient
	cfg              *config.Config
	publishEnabled   bool
	enqueuePublish   func(ctx context.Context, key string)
}

// NewSyncer builds the reconciler syncer. enqueuePublish feeds keys into the
// publisher controller's queue; it is a no-op func when marketplace
// publishing is disabled.
func NewSyncer(
	installerCache InstallerCache,
	marketplaceCache MarketplaceCache,
	storageClient StorageClient,
	downloader Downloader,
	statusClient StatusClient,
	cfg *config.Config,
	publishEnabled bool,
	enqueuePublish func(ctx context.Context, key string),
) *Syncer {
	return &Syncer{
		installerCache:   installerCache,
		marketplaceCache: marketplaceCache,
		storageClient:    storageClient,
		downloader:       downloader,
		statusClient:     statusClient,
		cfg:              cfg,
		publishEnabled:   publishEnabled,
		enqueuePublish:   enqueuePublish,
	}
}

// ParseKey splits a "{branch}/{arch}" workqueue key.
func ParseKey(key string) (branch, arch string, err error) {
	branch, arch, found := strings.Cut(key, "/")
	if !found || len(branch) == 0 || len(arch) == 0 {
		return "", "", fmt.Errorf("malformed reconcile key %q", key)
	}
	return branch, arch, nil
}

// SyncOnce implements the reconcile decision matrix for one
// branch/architecture pair.
func (s *Syncer) SyncOnce(ctx context.Context, key string) error {
	logger := utils.LoggerFromContext(ctx)

	branchName, arch, err := ParseKey(key)
	if err != nil {
		// Malformed keys can never succeed; drop them.
		logger.Error(err, "dropping malformed key")
		return nil
	}
	branch, configured := s.cfg.Branch(branchName)
	if !configured {
		logger.Info("branch no longer configured; ignoring", "branch", branchName)
		return nil
	}

	// Both listers must have a complete view before any stage/purge decision.
	if !s.installerCache.Synced() {
		return fmt.Errorf("installer lister has not completed a full sync yet")
	}
	if !s.marketplaceCache.Synced() {
		return fmt.Errorf("marketplace lister has not completed a full sync yet")
	}

	image, known := s.installerCache.Get(key)
	if !known {
		// The branch no longer pins an Azure image for this architecture:
		// purge anything staged and clear the status entry.
		if err := s.purgePrefix(ctx, branchName, arch, ""); err != nil {
			return err
		}
		return s.statusClient.UpdateArchStatus(ctx, branchName, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
			*archStatus = v1alpha1.RHCOSReleaseArchStatus{}
		})
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

	published := true
	for _, sku := range skus {
		if !s.marketplaceCache.HasVersion(sku.Name, version) {
			published = false
			break
		}
	}

	blobPath := storage.BlobPath(branchName, arch, image.Release)

	if published {
		// The marketplace serves the image; the staged VHD (and any older
		// leftovers) can go.
		if err := s.purgePrefix(ctx, branchName, arch, ""); err != nil {
			return err
		}
		logger.Info("image published in marketplace; staged VHD purged", "release", image.Release, "version", version)
		return s.statusClient.UpdateArchStatus(ctx, branchName, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
			archStatus.Release = image.Release
			archStatus.Phase = v1alpha1.ImagePhasePublished
			archStatus.StagedURL = ""
			archStatus.ConfigureJobID = ""
		})
	}

	staged, err := s.storageClient.BlobExists(ctx, blobPath)
	if err != nil {
		return err
	}
	if !staged {
		if err := s.stage(ctx, image, blobPath); err != nil {
			return err
		}
	}
	// Older releases of the same branch/arch are superseded; drop them.
	if err := s.purgePrefix(ctx, branchName, arch, blobPath); err != nil {
		return err
	}

	stagedURL, err := s.storageClient.WebURL(ctx, blobPath)
	if err != nil {
		return err
	}

	release, err := s.statusClient.Get(ctx, branchName)
	if err != nil {
		return err
	}
	currentStatus := release.Status.Architectures[arch]
	awaitingPublish := currentStatus.Phase == v1alpha1.ImagePhaseDraft && currentStatus.Release == image.Release

	if !awaitingPublish {
		if err := s.statusClient.UpdateArchStatus(ctx, branchName, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
			archStatus.Release = image.Release
			archStatus.Phase = v1alpha1.ImagePhaseStaged
			archStatus.StagedURL = stagedURL
			archStatus.ConfigureJobID = ""
		}); err != nil {
			return err
		}
		if s.publishEnabled {
			s.enqueuePublish(ctx, key)
		} else {
			logger.Info("VHD staged; marketplace publishing disabled", "release", image.Release, "stagedURL", stagedURL)
		}
		return nil
	}

	logger.Info("marketplace draft configured; awaiting manual publish", "release", image.Release, "version", version)
	return nil
}

// stage downloads, verifies and uploads the image's VHD, deleting the local
// copy afterwards.
func (s *Syncer) stage(ctx context.Context, image stream.Image, blobPath string) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Info("staging VHD", "release", image.Release, "url", image.DownloadURL)

	vhdPath, err := s.downloader.FetchVHD(ctx, image)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(vhdPath) }()

	stagedURL, err := s.storageClient.UploadFile(ctx, blobPath, vhdPath)
	if err != nil {
		return err
	}
	logger.Info("VHD staged", "release", image.Release, "stagedURL", stagedURL)
	return nil
}

// purgePrefix deletes all staged blobs of a branch/architecture pair except
// keep (pass empty to purge everything). Errors are accumulated so a
// transient failure on one blob does not block purging others.
func (s *Syncer) purgePrefix(ctx context.Context, branch, arch, keep string) error {
	logger := utils.LoggerFromContext(ctx)
	prefix := fmt.Sprintf("rhcos/%s/%s/", branch, arch)
	blobPaths, err := s.storageClient.ListBlobs(ctx, prefix)
	if err != nil {
		return err
	}
	var errs []error
	for _, blobPath := range blobPaths {
		if blobPath == keep {
			continue
		}
		if err := s.storageClient.DeleteBlob(ctx, blobPath); err != nil {
			errs = append(errs, err)
			continue
		}
		logger.Info("purged staged blob", "blob", blobPath)
	}
	return errors.Join(errs...)
}
