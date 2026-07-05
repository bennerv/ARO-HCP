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

package publisher

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/api/v1alpha1"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

type fakeInstallerCache struct {
	images map[string]stream.Image
}

func (f *fakeInstallerCache) Get(key string) (stream.Image, bool) {
	image, ok := f.images[key]
	return image, ok
}

type fakeStorage struct {
	blobs map[string]bool
}

func (f *fakeStorage) BlobExists(_ context.Context, blobPath string) (bool, error) {
	return f.blobs[blobPath], nil
}

func (f *fakeStorage) WebURL(_ context.Context, blobPath string) (string, error) {
	return "https://web.example.com/" + blobPath, nil
}

type fakeStatusClient struct {
	statuses map[string]*v1alpha1.RHCOSReleaseStatus
}

func (f *fakeStatusClient) Get(_ context.Context, branch string) (*v1alpha1.RHCOSRelease, error) {
	releaseStatus, ok := f.statuses[branch]
	if !ok {
		return nil, fmt.Errorf("RHCOSRelease %s not found", branch)
	}
	return &v1alpha1.RHCOSRelease{Status: *releaseStatus.DeepCopy()}, nil
}

func (f *fakeStatusClient) UpdateArchStatus(_ context.Context, branch, arch string, mutate func(*v1alpha1.RHCOSReleaseArchStatus)) error {
	releaseStatus := f.statuses[branch]
	if releaseStatus.Architectures == nil {
		releaseStatus.Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{}
	}
	archStatus := releaseStatus.Architectures[arch]
	mutate(&archStatus)
	releaseStatus.Architectures[arch] = archStatus
	return nil
}

// fakeIngestion simulates a product with a configurable set of pre-existing
// plans. Configure calls are recorded; plan-creation calls materialize the
// plan in the tree.
type fakeIngestion struct {
	productID     string
	plans         map[string]string         // externalID -> durable ID
	techConfigs   map[string]map[string]any // plan durable ID -> tech config
	configureLog  [][]map[string]any
	nextJob       int
	planIDCounter int
}

func (f *fakeIngestion) GetProductIDByExternalID(_ context.Context, _, _ string) (string, error) {
	return f.productID, nil
}

func (f *fakeIngestion) GetResourceTree(_ context.Context, _ string) (*marketplace.ResourceTree, error) {
	tree := &marketplace.ResourceTree{Root: f.productID}
	for externalID, planID := range f.plans {
		tree.Resources = append(tree.Resources, map[string]any{
			"$schema":  "https://schema.mp.microsoft.com/schema/plan/2022-03-01-preview3",
			"id":       planID,
			"identity": map[string]any{"externalId": externalID},
		})
	}
	for _, techConfig := range f.techConfigs {
		tree.Resources = append(tree.Resources, techConfig)
	}
	return tree, nil
}

func (f *fakeIngestion) Configure(_ context.Context, resources []map[string]any) (string, error) {
	f.configureLog = append(f.configureLog, resources)
	for _, resource := range resources {
		identity, _ := resource["identity"].(map[string]any)
		if _, isPlan := resource["product"]; isPlan && identity != nil && resource["plan"] == nil {
			externalID := identity["externalId"].(string)
			f.planIDCounter++
			f.plans[externalID] = fmt.Sprintf("plan/%s/generated-%d", f.productID, f.planIDCounter)
		}
	}
	f.nextJob++
	return fmt.Sprintf("job-%d", f.nextJob), nil
}

func (f *fakeIngestion) WaitForJob(_ context.Context, _ string) error {
	return nil
}

const (
	testBranch  = "release-4.22"
	testArch    = "x86_64"
	testRelease = "9.8.20260520-0"
	testKey     = testBranch + "/" + testArch
)

func newFixture(existingPlans map[string]string) (*Syncer, *fakeIngestion, *fakeStatusClient, *fakeStorage) {
	ingestion := &fakeIngestion{
		productID:   "product/aaa",
		plans:       existingPlans,
		techConfigs: map[string]map[string]any{},
	}
	statusClient := &fakeStatusClient{statuses: map[string]*v1alpha1.RHCOSReleaseStatus{
		testBranch: {},
	}}
	storageClient := &fakeStorage{blobs: map[string]bool{
		"rhcos/release-4.22/x86_64/rhcos-9.8.20260520-0-azure.x86_64.vhd": true,
	}}
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
	installerCache := &fakeInstallerCache{images: map[string]stream.Image{
		testKey: {Branch: testBranch, Arch: testArch, Release: testRelease},
	}}
	return NewSyncer(ingestion, installerCache, storageClient, statusClient, cfg), ingestion, statusClient, storageClient
}

func TestSyncOnceCreatesPlansAndDraft(t *testing.T) {
	syncer, ingestion, statusClient, _ := newFixture(map[string]string{})

	require.NoError(t, syncer.SyncOnce(t.Context(), testKey))

	// Two configure calls: plan creation, then tech configs.
	require.Len(t, ingestion.configureLog, 2)
	assert.Len(t, ingestion.configureLog[0], 2, "both x86 plans (Gen1+Gen2) created")
	assert.Len(t, ingestion.configureLog[1], 2, "both plans get a tech config")

	archStatus := statusClient.statuses[testBranch].Architectures[testArch]
	assert.Equal(t, v1alpha1.ImagePhaseDraft, archStatus.Phase)
	assert.Equal(t, testRelease, archStatus.Release)
	assert.Equal(t, "job-2", archStatus.ConfigureJobID)
}

func TestSyncOnceExistingPlansOnlyUpdatesTechConfig(t *testing.T) {
	syncer, ingestion, _, _ := newFixture(map[string]string{
		"aro_422-v2": "plan/aaa/gen2",
		"aro_422":    "plan/aaa/gen1",
	})

	require.NoError(t, syncer.SyncOnce(t.Context(), testKey))

	require.Len(t, ingestion.configureLog, 1, "no plan creation call")
	techConfigs := ingestion.configureLog[0]
	require.Len(t, techConfigs, 2)
	versions := techConfigs[0]["vmImageVersions"].([]any)
	require.Len(t, versions, 1)
	assert.Equal(t, "422.98.20260520", versions[0].(map[string]any)["versionNumber"])
}

func TestSyncOnceDraftAlreadyConfiguredIsNoOp(t *testing.T) {
	syncer, ingestion, statusClient, _ := newFixture(map[string]string{})
	statusClient.statuses[testBranch].Architectures = map[string]v1alpha1.RHCOSReleaseArchStatus{
		testArch: {Release: testRelease, Phase: v1alpha1.ImagePhaseDraft},
	}

	require.NoError(t, syncer.SyncOnce(t.Context(), testKey))
	assert.Empty(t, ingestion.configureLog)
}

func TestSyncOnceUnstagedVHDErrors(t *testing.T) {
	syncer, _, _, storageClient := newFixture(map[string]string{})
	storageClient.blobs = map[string]bool{}

	err := syncer.SyncOnce(t.Context(), testKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not staged")
}
