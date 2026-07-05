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

package installer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

type fakeStreamClient struct {
	images map[string][]stream.Image // key: branch
	errs   map[string]error
}

func (f *fakeStreamClient) FetchImages(_ context.Context, branch string, _ int) ([]stream.Image, error) {
	if err := f.errs[branch]; err != nil {
		return nil, err
	}
	return f.images[branch], nil
}

type fakeStatusClient struct {
	installerSyncs []string
}

func (f *fakeStatusClient) UpdateStatus(_ context.Context, branch string, rhelVersion int, mutate func(*v1alpha1.RHCOSReleaseStatus)) error {
	releaseStatus := &v1alpha1.RHCOSReleaseStatus{}
	mutate(releaseStatus)
	if releaseStatus.LastInstallerSync != nil {
		f.installerSyncs = append(f.installerSyncs, fmt.Sprintf("%s/rhel%d", branch, rhelVersion))
	}
	return nil
}

func testConfig() *config.Config {
	return &config.Config{
		Marketplace: config.Marketplace{Publisher: "azureopenshift", Offer: "aro4"},
		Branches: []config.Branch{
			{Name: "release-5.0", RHELVersions: []int{9}},
		},
	}
}

func testImage(branch string, rhelVersion int, arch, release string) stream.Image {
	return stream.Image{
		Branch:             branch,
		RHELVersion:        rhelVersion,
		Arch:               arch,
		Release:            release,
		DownloadURL:        fmt.Sprintf("https://rhcos.example.com/%s/%s.vhd.gz", release, arch),
		CompressedSHA256:   "aaa",
		UncompressedSHA256: "bbb",
	}
}

func TestSyncOnceEnqueuesNewImages(t *testing.T) {
	streamClient := &fakeStreamClient{images: map[string][]stream.Image{
		"release-5.0": {
			testImage("release-5.0", 9, "x86_64", "9.8.20260520-0"),
			testImage("release-5.0", 9, "aarch64", "9.8.20260520-0"),
		},
	}}
	statusClient := &fakeStatusClient{}
	var enqueued []string
	syncer := NewSyncer(streamClient, statusClient, testConfig(), func(_ context.Context, key string) { enqueued = append(enqueued, key) })

	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))

	assert.ElementsMatch(t, []string{"release-5.0/rhel9/x86_64", "release-5.0/rhel9/aarch64"}, enqueued)
	assert.True(t, syncer.Synced())
	assert.Equal(t, []string{"release-5.0/rhel9"}, statusClient.installerSyncs)

	image, ok := syncer.Get("release-5.0/rhel9/x86_64")
	require.True(t, ok)
	assert.Equal(t, "9.8.20260520-0", image.Release)
}

func TestSyncOnceEnqueuesOnlyChanges(t *testing.T) {
	streamClient := &fakeStreamClient{images: map[string][]stream.Image{
		"release-5.0": {
			testImage("release-5.0", 9, "x86_64", "9.8.20260520-0"),
			testImage("release-5.0", 9, "aarch64", "9.8.20260520-0"),
		},
	}}
	var enqueued []string
	syncer := NewSyncer(streamClient, &fakeStatusClient{}, testConfig(), func(_ context.Context, key string) { enqueued = append(enqueued, key) })
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))

	// Second sync with identical data: nothing enqueued.
	enqueued = nil
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.Empty(t, enqueued)

	// New release for one arch: only that key enqueued.
	streamClient.images["release-5.0"] = []stream.Image{
		testImage("release-5.0", 9, "x86_64", "9.8.20260621-0"),
		testImage("release-5.0", 9, "aarch64", "9.8.20260520-0"),
	}
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.Equal(t, []string{"release-5.0/rhel9/x86_64"}, enqueued)
}

func TestSyncOnceEnqueuesRemovals(t *testing.T) {
	streamClient := &fakeStreamClient{images: map[string][]stream.Image{
		"release-5.0": {
			testImage("release-5.0", 9, "x86_64", "9.8.20260520-0"),
			testImage("release-5.0", 9, "aarch64", "9.8.20260520-0"),
		},
	}}
	var enqueued []string
	syncer := NewSyncer(streamClient, &fakeStatusClient{}, testConfig(), func(_ context.Context, key string) { enqueued = append(enqueued, key) })
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))

	enqueued = nil
	streamClient.images["release-5.0"] = []stream.Image{
		testImage("release-5.0", 9, "x86_64", "9.8.20260520-0"),
	}
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.Equal(t, []string{"release-5.0/rhel9/aarch64"}, enqueued)

	_, ok := syncer.Get("release-5.0/rhel9/aarch64")
	assert.False(t, ok)
}

func TestSyncOnceKeepsCacheOnFetchError(t *testing.T) {
	streamClient := &fakeStreamClient{images: map[string][]stream.Image{
		"release-5.0": {testImage("release-5.0", 9, "x86_64", "9.8.20260520-0")},
	}}
	syncer := NewSyncer(streamClient, &fakeStatusClient{}, testConfig(), func(context.Context, string) {})
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.True(t, syncer.Synced())

	streamClient.errs = map[string]error{"release-5.0": fmt.Errorf("github down")}
	err := syncer.SyncOnce(t.Context(), SyncKey)
	require.Error(t, err)

	// The cached image survives the failed poll.
	image, ok := syncer.Get("release-5.0/rhel9/x86_64")
	require.True(t, ok)
	assert.Equal(t, "9.8.20260520-0", image.Release)
}

func TestSyncedRequiresFullSuccess(t *testing.T) {
	streamClient := &fakeStreamClient{
		errs: map[string]error{"release-5.0": fmt.Errorf("github down")},
	}
	syncer := NewSyncer(streamClient, &fakeStatusClient{}, testConfig(), func(context.Context, string) {})
	require.Error(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.False(t, syncer.Synced())
}
