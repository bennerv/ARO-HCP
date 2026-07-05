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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTree(t *testing.T) *ResourceTree {
	t.Helper()
	raw := `{
  "root": "product/aaa",
  "resources": [
    {
      "$schema": "https://schema.mp.microsoft.com/schema/plan/2022-03-01-preview3",
      "id": "plan/aaa/bbb",
      "identity": {"externalId": "aro_422-v2"},
      "alias": "aro_422-v2"
    },
    {
      "$schema": "https://schema.mp.microsoft.com/schema/virtual-machine-plan-technical-configuration/2022-03-01-preview3",
      "id": "virtual-machine-plan-technical-configuration/aaa/bbb",
      "plan": "plan/aaa/bbb",
      "operatingSystem": {"family": "linux"},
      "unmodeledField": {"must": "survive"},
      "skus": [
        {"skuId": "aro_422-v2", "imageType": "x64Gen2", "securityType": ["trusted"]}
      ],
      "vmImageVersions": [
        {"versionNumber": "422.98.20260101", "vmImages": [{"imageType": "x64Gen2", "source": {"sourceType": "sasUri", "osDisk": {"uri": "https://old"}}}]}
      ]
    }
  ]
}`
	tree := &ResourceTree{}
	require.NoError(t, json.Unmarshal([]byte(raw), tree))
	return tree
}

func TestFindPlanDurableID(t *testing.T) {
	tree := testTree(t)
	planID, found := FindPlanDurableID(tree, "aro_422-v2")
	require.True(t, found)
	assert.Equal(t, "plan/aaa/bbb", planID)

	_, found = FindPlanDurableID(tree, "aro_423-v2")
	assert.False(t, found)
}

func TestFindTechConfig(t *testing.T) {
	tree := testTree(t)
	techConfig, found := FindTechConfig(tree, "plan/aaa/bbb")
	require.True(t, found)
	assert.Equal(t, "virtual-machine-plan-technical-configuration/aaa/bbb", techConfig["id"])

	_, found = FindTechConfig(tree, "plan/aaa/ccc")
	assert.False(t, found)
}

func TestEnsureSKUs(t *testing.T) {
	tree := testTree(t)
	techConfig, _ := FindTechConfig(tree, "plan/aaa/bbb")

	// Existing SKU: no change.
	changed := EnsureSKUs(techConfig, []SKU{{Name: "aro_422-v2", ImageType: ImageTypeX64Gen2, SecurityTypes: []string{"trusted"}}})
	assert.False(t, changed)

	// New SKU appended, existing entries untouched.
	changed = EnsureSKUs(techConfig, []SKU{{Name: "aro_422", ImageType: ImageTypeX64Gen1}})
	assert.True(t, changed)
	skus := techConfig["skus"].([]any)
	require.Len(t, skus, 2)
	assert.Equal(t, "aro_422", skus[1].(map[string]any)["skuId"])
	_, hasSecurityType := skus[1].(map[string]any)["securityType"]
	assert.False(t, hasSecurityType, "Gen1 SKUs carry no security type")
}

func TestEnsureImageVersion(t *testing.T) {
	tree := testTree(t)
	techConfig, _ := FindTechConfig(tree, "plan/aaa/bbb")

	// Existing version: never modified.
	changed := EnsureImageVersion(techConfig, "422.98.20260101", []VMImage{{ImageType: ImageTypeX64Gen2, URI: "https://new"}})
	assert.False(t, changed)

	changed = EnsureImageVersion(techConfig, "422.98.20260520", []VMImage{{ImageType: ImageTypeX64Gen2, URI: "https://web.example.com/x.vhd"}})
	assert.True(t, changed)

	versions := techConfig["vmImageVersions"].([]any)
	require.Len(t, versions, 2)
	added := versions[1].(map[string]any)
	assert.Equal(t, "422.98.20260520", added["versionNumber"])
	vmImages := added["vmImages"].([]any)
	require.Len(t, vmImages, 1)
	source := vmImages[0].(map[string]any)["source"].(map[string]any)
	assert.Equal(t, "sasUri", source["sourceType"])
	assert.Equal(t, "https://web.example.com/x.vhd", source["osDisk"].(map[string]any)["uri"])

	// Unmodeled fields survive the modification round trip.
	assert.Equal(t, map[string]any{"must": "survive"}, techConfig["unmodeledField"])
}

func TestNewPlanResource(t *testing.T) {
	plan := NewPlanResource("product/aaa", "aro_423-v2", "aro_423-v2")
	assert.Equal(t, "product/aaa", plan["product"])
	assert.Equal(t, "aro_423-v2", plan["identity"].(map[string]any)["externalId"])
	assert.Equal(t, []any{"azureGlobal"}, plan["azureRegions"])
}
