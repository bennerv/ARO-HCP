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
	"strconv"
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
	Get(ctx context.Context, branch string, rhelVersion int) (*v1alpha1.RHCOSRelease, error)
	UpdateArchStatus(ctx context.Context, branch string, rhelVersion int, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error
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

// ParseKey splits a "{branch}/rhel{N}/{arch}" workqueue key.
func ParseKey(key string) (branch string, rhelVersion int, arch string, err error) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) != 3 || len(parts[0]) == 0 || len(parts[2]) == 0 {
		return "", 0, "", fmt.Errorf("malformed reconcile key %q", key)
	}
	rhel := parts[1]
	if !strings.HasPrefix(rhel, "rhel") {
		return "", 0, "", fmt.Errorf("malformed reconcile key %q: expected rhel<N> segment", key)
	}
	rhelVersion, err = strconv.Atoi(rhel[4:])
	if err != nil {
		return "", 0, "", fmt.Errorf("malformed reconcile key %q: %w", key, err)
	}
	return parts[0], rhelVersion, parts[2], nil
}

// SyncOnce implements the reconcile decision matrix for one
// branch/architecture pair.
func (s *Syncer) SyncOnce(ctx context.Context, key string) error {
	logger := utils.LoggerFromContext(ctx)

	branchName, rhelVersion, arch, err := ParseKey(key)
	if err != nil {
		logger.Error(err, "dropping malformed key")
		return nil
	}
	branch, configured := s.cfg.Branch(branchName)
	if !configured {
		logger.Info("branch no longer configured; ignoring", "branch", branchName)
		return nil
	}

	if !s.installerCache.Synced() {
		return fmt.Errorf("installer lister has not completed a full sync yet")
	}
	if !s.marketplaceCache.Synced() {
		return fmt.Errorf("marketplace lister has not completed a full sync yet")
	}

	image, known := s.installerCache.Get(key)
	if !known {
		if err := s.purgePrefix(ctx, branchName, rhelVersion, arch, ""); err != nil {
			return err
		}
		return s.statusClient.UpdateArchStatus(ctx, branchName, rhelVersion, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
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

	blobPath := storage.BlobPath(branchName, rhelVersion, arch, image.Release)

	if published {
		if err := s.purgePrefix(ctx, branchName, rhelVersion, arch, ""); err != nil {
			return err
		}
		logger.Info("image published in marketplace; staged VHD purged", "release", image.Release, "version", version)
		return s.statusClient.UpdateArchStatus(ctx, branchName, rhelVersion, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
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
	if err := s.purgePrefix(ctx, branchName, rhelVersion, arch, blobPath); err != nil {
		return err
	}

	stagedURL, err := s.storageClient.WebURL(ctx, blobPath)
	if err != nil {
		return err
	}

	release, err := s.statusClient.Get(ctx, branchName, rhelVersion)
	if err != nil {
		return err
	}
	currentStatus := release.Status.Architectures[arch]
	awaitingPublish := currentStatus.Phase == v1alpha1.ImagePhaseDraft && currentStatus.Release == image.Release

	if !awaitingPublish {
		if err := s.statusClient.UpdateArchStatus(ctx, branchName, rhelVersion, arch, func(archStatus *v1alpha1.RHCOSReleaseArchStatus) {
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

// stage downloads, verifies and uploads the image's VHD. The local copy is
// kept on failure so a requeue can reuse it (the PVC survives restarts);
// it is deleted only after a successful upload.
func (s *Syncer) stage(ctx context.Context, image stream.Image, blobPath string) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Info("staging VHD", "release", image.Release, "url", image.DownloadURL)

	vhdPath, err := s.downloader.FetchVHD(ctx, image)
	if err != nil {
		return err
	}

	stagedURL, err := s.storageClient.UploadFile(ctx, blobPath, vhdPath)
	if err != nil {
		return err
	}

	_ = os.Remove(vhdPath)
	logger.Info("VHD staged", "release", image.Release, "stagedURL", stagedURL)
	return nil
}

// purgePrefix deletes all staged blobs of a branch/architecture pair except
// keep (pass empty to purge everything). Errors are accumulated so a
// transient failure on one blob does not block purging others.
func (s *Syncer) purgePrefix(ctx context.Context, branch string, rhelVersion int, arch, keep string) error {
	logger := utils.LoggerFromContext(ctx)
	prefix := storage.BlobPrefix(branch, rhelVersion, arch)
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
