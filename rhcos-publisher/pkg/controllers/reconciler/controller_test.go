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

package reconciler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/storage"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

type fakeInstallerCache struct {
	images map[string]stream.Image
	synced bool
}

func (f *fakeInstallerCache) Get(key string) (stream.Image, bool) {
	image, ok := f.images[key]
	return image, ok
}
func (f *fakeInstallerCache) Synced() bool { return f.synced }

type fakeMarketplaceCache struct {
	versions map[string][]string // key: SKU
	synced   bool
}

func (f *fakeMarketplaceCache) HasVersion(sku, version string) bool {
	for _, v := range f.versions[sku] {
		if v == version {
			return true
		}
	}
	return false
}
func (f *fakeMarketplaceCache) Synced() bool { return f.synced }

type fakeStorage struct {
	blobs    map[string]bool
	uploaded []string
	deleted  []string
}

func (f *fakeStorage) BlobExists(_ context.Context, blobPath string) (bool, error) {
	return f.blobs[blobPath], nil
}
func (f *fakeStorage) UploadFile(_ context.Context, blobPath, _ string) (string, error) {
	f.blobs[blobPath] = true
	f.uploaded = append(f.uploaded, blobPath)
	return "https://web.example.com/" + blobPath, nil
}
func (f *fakeStorage) DeleteBlob(_ context.Context, blobPath string) error {
	delete(f.blobs, blobPath)
	f.deleted = append(f.deleted, blobPath)
	return nil
}
func (f *fakeStorage) ListBlobs(_ context.Context, prefix string) ([]string, error) {
	var paths []string
	for blobPath := range f.blobs {
		if strings.HasPrefix(blobPath, prefix) {
			paths = append(paths, blobPath)
		}
	}
	return paths, nil
}
func (f *fakeStorage) WebURL(_ context.Context, blobPath string) (string, error) {
	return "https://web.example.com/" + blobPath, nil
}

type fakeDownloader struct {
	fetched []string
	err     error
}

func (f *fakeDownloader) FetchVHD(_ context.Context, image stream.Image) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.fetched = append(f.fetched, image.Key())
	return "/tmp/fake.vhd", nil
}

type fakeStatusClient struct {
	statuses map[string]*v1alpha1.RHCOSReleaseStatus // key: branch
}

func newFakeStatusClient(branches ...string) *fakeStatusClient {
	statuses := map[string]*v1alpha1.RHCOSReleaseStatus{}
	for _, branch := range branches {
		statuses[branch] = &v1alpha1.RHCOSReleaseStatus{}
	}
	return &fakeStatusClient{statuses: statuses}
}

func (f *fakeStatusClient) Get(_ context.Context, branch string) (*v1alpha1.RHCOSRelease, error) {
	releaseStatus, ok := f.statuses[branch]
	if !ok {
		return nil, fmt.Errorf("RHCOSRelease %s not found", branch)
	}
	return &v1alpha1.RHCOSRelease{Status: *releaseStatus.DeepCopy()}, nil
}

func (f *fakeStatusClient) UpdateArchStatus(_ context.Context, branch, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error {
	releaseStatus, ok := f.statuses[branch]
	if !ok {
		return fmt.Errorf("RHCOSRelease %s not found", branch)
	}
	if releaseStatus.Architectures == nil {
		releaseStatus.Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{}
	}
	archStatus := releaseStatus.Architectures[arch]
	mutate(&archStatus)
	releaseStatus.Architectures[arch] = archStatus
	return nil
}

const (
	testBranch  = "release-4.22"
	testArch    = "x86_64"
	testRelease = "9.8.20260520-0"
	testVersion = "422.98.20260520"
	testKey     = testBranch + "/" + testArch
)

var testBlobPath = storage.BlobPath(testBranch, testArch, testRelease)

type fixture struct {
	installerCache   *fakeInstallerCache
	marketplaceCache *fakeMarketplaceCache
	storageClient    *fakeStorage
	downloader       *fakeDownloader
	statusClient     *fakeStatusClient
	published        []string
	syncer           *Syncer
}

func newFixture(publishEnabled bool) *fixture {
	f := &fixture{
		installerCache: &fakeInstallerCache{
			images: map[string]stream.Image{
				testKey: {
					Branch:             testBranch,
					Arch:               testArch,
					Release:            testRelease,
					DownloadURL:        "https://rhcos.example.com/x.vhd.gz",
					CompressedSHA256:   "aaa",
					UncompressedSHA256: "bbb",
				},
			},
			synced: true,
		},
		marketplaceCache: &fakeMarketplaceCache{versions: map[string][]string{}, synced: true},
		storageClient:    &fakeStorage{blobs: map[string]bool{}},
		downloader:       &fakeDownloader{},
		statusClient:     newFakeStatusClient(testBranch),
	}
	cfg := &config.Config{
		Marketplace: config.Marketplace{Publisher: "azureopenshift", Offer: "aro4"},
		Branches: []config.Branch{
			{
				Name:        testBranch,
				RHELVersion: 9,
				X86Features: []string{config.FeatureTrustedLaunch, config.FeatureHyperVGen1},
				ARMFeatures: []string{config.FeatureTrustedLaunch},
			},
		},
	}
	f.syncer = NewSyncer(
		f.installerCache, f.marketplaceCache, f.storageClient, f.downloader, f.statusClient,
		cfg, publishEnabled, func(_ context.Context, key string) { f.published = append(f.published, key) },
	)
	return f
}

func (f *fixture) archStatus(t *testing.T) v1alpha1.RHCOSReleaseArchStatus {
	t.Helper()
	return f.statusClient.statuses[testBranch].Architectures[testArch]
}

func TestNewReleaseNotInMarketplaceStagesVHD(t *testing.T) {
	f := newFixture(true)
	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))

	assert.Equal(t, []string{testKey}, f.downloader.fetched)
	assert.Equal(t, []string{testBlobPath}, f.storageClient.uploaded)
	assert.Equal(t, []string{testKey}, f.published, "publisher must be enqueued after staging")

	archStatus := f.archStatus(t)
	assert.Equal(t, v1alpha1.ImagePhaseStaged, archStatus.Phase)
	assert.Equal(t, testRelease, archStatus.Release)
	assert.Equal(t, "https://web.example.com/"+testBlobPath, archStatus.StagedURL)
}

func TestStagedPublishDisabledIsNoOp(t *testing.T) {
	f := newFixture(false)
	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))

	assert.Equal(t, []string{testBlobPath}, f.storageClient.uploaded)
	assert.Empty(t, f.published, "publisher must not be enqueued when publishing is disabled")
	assert.Equal(t, v1alpha1.ImagePhaseStaged, f.archStatus(t).Phase)
}

func TestAlreadyStagedDoesNotRedownload(t *testing.T) {
	f := newFixture(true)
	f.storageClient.blobs[testBlobPath] = true

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Empty(t, f.downloader.fetched)
	assert.Empty(t, f.storageClient.uploaded)
	assert.Equal(t, []string{testKey}, f.published)
}

func TestDraftAwaitingManualPublishIsNoOp(t *testing.T) {
	f := newFixture(true)
	f.storageClient.blobs[testBlobPath] = true
	f.statusClient.statuses[testBranch].Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{
		testArch: {Release: testRelease, Phase: v1alpha1.ImagePhaseDraft, ConfigureJobID: "job-1"},
	}

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Empty(t, f.published, "a configured draft awaits a human publish")
	assert.Equal(t, v1alpha1.ImagePhaseDraft, f.archStatus(t).Phase)
}

func TestPublishedImagePurgesStagedVHD(t *testing.T) {
	f := newFixture(true)
	f.storageClient.blobs[testBlobPath] = true
	f.marketplaceCache.versions = map[string][]string{
		"aro_422-v2": {testVersion},
		"aro_422":    {testVersion},
	}

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Equal(t, []string{testBlobPath}, f.storageClient.deleted)
	assert.Empty(t, f.published)

	archStatus := f.archStatus(t)
	assert.Equal(t, v1alpha1.ImagePhasePublished, archStatus.Phase)
	assert.Empty(t, archStatus.StagedURL)
}

func TestPartiallyPublishedStillStages(t *testing.T) {
	// Only the Gen2 SKU carries the version; Gen1 is still missing, so the
	// image is not considered published.
	f := newFixture(true)
	f.marketplaceCache.versions = map[string][]string{
		"aro_422-v2": {testVersion},
	}

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Equal(t, []string{testBlobPath}, f.storageClient.uploaded)
	assert.Equal(t, v1alpha1.ImagePhaseStaged, f.archStatus(t).Phase)
}

func TestSupersededReleaseIsPurged(t *testing.T) {
	f := newFixture(true)
	oldBlobPath := storage.BlobPath(testBranch, testArch, "9.8.20260101-0")
	f.storageClient.blobs[oldBlobPath] = true

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Contains(t, f.storageClient.deleted, oldBlobPath)
	assert.True(t, f.storageClient.blobs[testBlobPath])
}

func TestImageRemovedFromStreamCleansUp(t *testing.T) {
	f := newFixture(true)
	delete(f.installerCache.images, testKey)
	f.storageClient.blobs[testBlobPath] = true
	f.statusClient.statuses[testBranch].Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{
		testArch: {Release: testRelease, Phase: v1alpha1.ImagePhaseStaged},
	}

	require.NoError(t, f.syncer.SyncOnce(t.Context(), testKey))
	assert.Equal(t, []string{testBlobPath}, f.storageClient.deleted)
	assert.Equal(t, v1alpha1.RHCOSReleaseArchStatus{}, f.archStatus(t))
}

func TestUnsyncedCachesError(t *testing.T) {
	f := newFixture(true)
	f.marketplaceCache.synced = false
	require.Error(t, f.syncer.SyncOnce(t.Context(), testKey))

	f = newFixture(true)
	f.installerCache.synced = false
	require.Error(t, f.syncer.SyncOnce(t.Context(), testKey))
}

func TestUnconfiguredBranchIsIgnored(t *testing.T) {
	f := newFixture(true)
	require.NoError(t, f.syncer.SyncOnce(t.Context(), "release-4.19/x86_64"))
	assert.Empty(t, f.downloader.fetched)
}

func TestMalformedKeyIsDropped(t *testing.T) {
	f := newFixture(true)
	require.NoError(t, f.syncer.SyncOnce(t.Context(), "garbage"))
}

func TestParseKey(t *testing.T) {
	branch, arch, err := ParseKey("release-4.22/x86_64")
	require.NoError(t, err)
	assert.Equal(t, "release-4.22", branch)
	assert.Equal(t, "x86_64", arch)

	_, _, err = ParseKey("garbage")
	assert.Error(t, err)
}
