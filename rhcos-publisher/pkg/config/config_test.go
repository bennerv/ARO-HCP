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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validConfig = `
marketplace:
  publisher: azureopenshift
  offer: aro4
branches:
- name: release-5.0
  rhelVersions:
  - 9
  - 10
  armFeatures:
  - TrustedLaunchSupported
  x86Features:
  - TrustedLaunchSupported
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoad(t *testing.T) {
	cfg, err := Load(writeConfig(t, validConfig))
	require.NoError(t, err)
	assert.Equal(t, "azureopenshift", cfg.Marketplace.Publisher)
	assert.Equal(t, "aro4", cfg.Marketplace.Offer)
	require.Len(t, cfg.Branches, 1)
	assert.Equal(t, "release-5.0", cfg.Branches[0].Name)
	assert.Equal(t, []int{9, 10}, cfg.Branches[0].RHELVersions)
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	_, err := Load(writeConfig(t, validConfig+"\nunknownField: true\n"))
	assert.Error(t, err)
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mutate      func(*Config)
		expectError string
	}{
		{
			name:   "valid",
			mutate: func(c *Config) {},
		},
		{
			name:        "missing publisher",
			mutate:      func(c *Config) { c.Marketplace.Publisher = "" },
			expectError: "marketplace.publisher is required",
		},
		{
			name:        "missing offer",
			mutate:      func(c *Config) { c.Marketplace.Offer = "" },
			expectError: "marketplace.offer is required",
		},
		{
			name:        "no branches",
			mutate:      func(c *Config) { c.Branches = nil },
			expectError: "at least one branch is required",
		},
		{
			name:        "duplicate branch",
			mutate:      func(c *Config) { c.Branches = append(c.Branches, c.Branches[0]) },
			expectError: "duplicate branch",
		},
		{
			name:        "malformed branch name",
			mutate:      func(c *Config) { c.Branches[0].Name = "main" },
			expectError: "does not match release-<major>.<minor>",
		},
		{
			name:        "empty rhel versions",
			mutate:      func(c *Config) { c.Branches[0].RHELVersions = nil },
			expectError: "at least one rhelVersion is required",
		},
		{
			name:        "zero rhel version",
			mutate:      func(c *Config) { c.Branches[0].RHELVersions = []int{0} },
			expectError: "rhelVersions must be positive",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Marketplace: Marketplace{Publisher: "azureopenshift", Offer: "aro4"},
				Branches: []Branch{
					{Name: "release-5.0", RHELVersions: []int{9}},
				},
			}
			tc.mutate(cfg)
			err := cfg.Validate()
			if len(tc.expectError) == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			}
		})
	}
}

func TestMinorID(t *testing.T) {
	minorID, err := Branch{Name: "release-4.22"}.MinorID()
	require.NoError(t, err)
	assert.Equal(t, "422", minorID)

	minorID, err = Branch{Name: "release-5.0"}.MinorID()
	require.NoError(t, err)
	assert.Equal(t, "50", minorID)

	_, err = Branch{Name: "release-4.22.1"}.MinorID()
	assert.Error(t, err)
}

func TestVersion(t *testing.T) {
	v, err := Branch{Name: "release-4.22"}.Version()
	require.NoError(t, err)
	assert.Equal(t, uint64(4), v.Major)
	assert.Equal(t, uint64(22), v.Minor)

	v, err = Branch{Name: "release-5.0"}.Version()
	require.NoError(t, err)
	assert.Equal(t, uint64(5), v.Major)
	assert.Equal(t, uint64(0), v.Minor)

	_, err = Branch{Name: "main"}.Version()
	assert.Error(t, err)
}

func TestFeatures(t *testing.T) {
	branch := Branch{
		Name:        "release-5.0",
		ARMFeatures: []string{"a"},
		X86Features: []string{"b"},
	}
	features, err := branch.Features(ArchAarch64)
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, features)

	features, err = branch.Features(ArchX86_64)
	require.NoError(t, err)
	assert.Equal(t, []string{"b"}, features)

	_, err = branch.Features("s390x")
	assert.Error(t, err)
}
