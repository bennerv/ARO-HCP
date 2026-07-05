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

package marketplace

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
)

type fakeVersionLister struct {
	versions map[string][]string // key: SKU
	errs     map[string]error
}

func (f *fakeVersionLister) ListVersions(_ context.Context, _, _, sku string) ([]string, error) {
	if err := f.errs[sku]; err != nil {
		return nil, err
	}
	return f.versions[sku], nil
}

type fakeStatusClient struct {
	marketplaceSyncs []string
}

func (f *fakeStatusClient) UpdateStatus(_ context.Context, branch string, rhelVersion int, mutate func(*v1alpha1.RHCOSReleaseStatus)) error {
	releaseStatus := &v1alpha1.RHCOSReleaseStatus{}
	mutate(releaseStatus)
	if releaseStatus.LastMarketplaceSync != nil {
		f.marketplaceSyncs = append(f.marketplaceSyncs, fmt.Sprintf("%s/rhel%d", branch, rhelVersion))
	}
	return nil
}

func testConfig() *config.Config {
	return &config.Config{
		Marketplace: config.Marketplace{Publisher: "azureopenshift", Offer: "aro4"},
		Branches: []config.Branch{
			{
				Name:         "release-5.0",
				RHELVersions: []int{9},
				ARMFeatures:  []string{config.FeatureTrustedLaunch},
				X86Features:  []string{config.FeatureTrustedLaunch},
			},
		},
	}
}

func TestSyncOnceCachesVersionsAndEnqueuesChanges(t *testing.T) {
	versionLister := &fakeVersionLister{versions: map[string][]string{
		"aro_50-x64":  {"50.98.20260101"},
		"aro_50-arm": {},
	}}
	statusClient := &fakeStatusClient{}
	var enqueued []string
	syncer := NewSyncer(versionLister, statusClient, testConfig(), func(_ context.Context, key string) { enqueued = append(enqueued, key) })

	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	// First sync populates the cache: every branch/arch is "changed".
	assert.ElementsMatch(t, []string{"release-5.0/rhel9/x86_64", "release-5.0/rhel9/aarch64"}, enqueued)
	assert.True(t, syncer.Synced())
	assert.Equal(t, []string{"release-5.0/rhel9"}, statusClient.marketplaceSyncs)
	assert.True(t, syncer.HasVersion("aro_50-x64", "50.98.20260101"))
	assert.False(t, syncer.HasVersion("aro_50-arm", "50.98.20260101"))

	// Unchanged second sync: nothing enqueued.
	enqueued = nil
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.Empty(t, enqueued)

	// A new ARM version appears: only aarch64 enqueued.
	versionLister.versions["aro_50-arm"] = []string{"50.98.20260101"}
	require.NoError(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.Equal(t, []string{"release-5.0/rhel9/aarch64"}, enqueued)
	assert.True(t, syncer.HasVersion("aro_50-arm", "50.98.20260101"))
}

func TestSyncOnceListError(t *testing.T) {
	versionLister := &fakeVersionLister{
		versions: map[string][]string{},
		errs:     map[string]error{"aro_50-x64": fmt.Errorf("throttled")},
	}
	statusClient := &fakeStatusClient{}
	syncer := NewSyncer(versionLister, statusClient, testConfig(), func(context.Context, string) {})

	require.Error(t, syncer.SyncOnce(t.Context(), SyncKey))
	assert.False(t, syncer.Synced())
	// The branch had a failing SKU; its sync timestamp must not advance.
	assert.Empty(t, statusClient.marketplaceSyncs)
}
