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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
)

func TestSKUsForArchX86(t *testing.T) {
	branch := config.Branch{
		Name:        "release-4.22",
		X86Features: []string{config.FeatureTrustedLaunch, config.FeatureHyperVGen1},
	}
	skus, err := SKUsForArch(branch, config.ArchX86_64)
	require.NoError(t, err)
	require.Len(t, skus, 2)
	assert.Equal(t, SKU{Name: "aro_422-v2", ImageType: ImageTypeX64Gen2, SecurityTypes: []string{"trusted"}}, skus[0])
	assert.Equal(t, SKU{Name: "aro_422", ImageType: ImageTypeX64Gen1}, skus[1])
}

func TestSKUsForArchX86WithoutGen1(t *testing.T) {
	branch := config.Branch{
		Name:        "release-4.22",
		X86Features: []string{config.FeatureTrustedLaunch},
	}
	skus, err := SKUsForArch(branch, config.ArchX86_64)
	require.NoError(t, err)
	require.Len(t, skus, 1)
	assert.Equal(t, "aro_422-v2", skus[0].Name)
}

func TestSKUsForArchARM(t *testing.T) {
	branch := config.Branch{
		Name:        "release-4.22",
		ARMFeatures: []string{config.FeatureTrustedLaunch},
	}
	skus, err := SKUsForArch(branch, config.ArchAarch64)
	require.NoError(t, err)
	require.Len(t, skus, 1)
	assert.Equal(t, SKU{Name: "aro_422-arm", ImageType: ImageTypeARM64Gen2, SecurityTypes: []string{"trusted"}}, skus[0])
}

func TestSKUsForArchUnknown(t *testing.T) {
	_, err := SKUsForArch(config.Branch{Name: "release-4.22"}, "s390x")
	assert.Error(t, err)
}

func TestImageVersion(t *testing.T) {
	version, err := ImageVersion("422", "9.8.20260520-0")
	require.NoError(t, err)
	assert.Equal(t, "422.98.20260520", version)

	// Release timestamps with more than a date component keep only the date.
	version, err = ImageVersion("420", "9.6.202505201234-0")
	require.NoError(t, err)
	assert.Equal(t, "420.96.20250520", version)

	_, err = ImageVersion("422", "not-a-release")
	assert.Error(t, err)
}
