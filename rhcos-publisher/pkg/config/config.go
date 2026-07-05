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

// Package config holds the file-based configuration of the rhcos-publisher
// controller. The file is rendered from the ARO-HCP configuration system into
// a ConfigMap and mounted into the controller pod; a content change triggers
// a graceful process restart.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Architectures supported by the ARO marketplace offering. These are coreos
// stream architecture names.
const (
	ArchX86_64  = "x86_64"
	ArchAarch64 = "aarch64"
)

// Marketplace plan feature names understood by the publisher.
const (
	// FeatureTrustedLaunch marks Gen2 SKUs as supporting Trusted Launch.
	FeatureTrustedLaunch = "TrustedLaunchSupported"
	// FeatureHyperVGen1 requests an additional Hyper-V generation 1 SKU
	// (x86_64 only; ARM images are always Gen2).
	FeatureHyperVGen1 = "HyperVGeneration.V1"
)

var branchNameRE = regexp.MustCompile(`^release-(\d+)\.(\d+)$`)

// Config is the root of the mounted configuration file.
type Config struct {
	// Marketplace identifies the ARO offering to publish to.
	Marketplace Marketplace `yaml:"marketplace"`
	// Branches are the OCP release branches to mirror.
	Branches []Branch `yaml:"branches"`
}

// Marketplace identifies a marketplace offering.
type Marketplace struct {
	// Publisher is the marketplace publisher ID, e.g. "azureopenshift".
	Publisher string `yaml:"publisher"`
	// Offer is the marketplace offer ID, e.g. "aro4".
	Offer string `yaml:"offer"`
}

// Branch configures one OCP release branch.
type Branch struct {
	// Name is the openshift/installer branch name, e.g. "release-4.22".
	Name string `yaml:"name"`
	// RHELVersion selects the coreos stream file, e.g. 9 for coreos-rhel-9.json.
	RHELVersion int `yaml:"rhelVersion"`
	// ARMFeatures are marketplace plan features of the aarch64 images.
	ARMFeatures []string `yaml:"armFeatures"`
	// X86Features are marketplace plan features of the x86_64 images.
	X86Features []string `yaml:"x86Features"`
}

// Load reads and validates the configuration file at path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfg := &Config{}
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks the configuration for structural errors.
func (c *Config) Validate() error {
	var errs []error
	if len(c.Marketplace.Publisher) == 0 {
		errs = append(errs, errors.New("marketplace.publisher is required"))
	}
	if len(c.Marketplace.Offer) == 0 {
		errs = append(errs, errors.New("marketplace.offer is required"))
	}
	if len(c.Branches) == 0 {
		errs = append(errs, errors.New("at least one branch is required"))
	}
	seen := map[string]bool{}
	for i, branch := range c.Branches {
		if seen[branch.Name] {
			errs = append(errs, fmt.Errorf("branches[%d]: duplicate branch %q", i, branch.Name))
		}
		seen[branch.Name] = true
		if _, err := branch.MinorID(); err != nil {
			errs = append(errs, fmt.Errorf("branches[%d]: %w", i, err))
		}
		if branch.RHELVersion <= 0 {
			errs = append(errs, fmt.Errorf("branches[%d]: rhelVersion must be positive", i))
		}
	}
	return errors.Join(errs...)
}

// Branch returns the configuration of the named branch.
func (c *Config) Branch(name string) (Branch, bool) {
	for _, branch := range c.Branches {
		if branch.Name == name {
			return branch, true
		}
	}
	return Branch{}, false
}

// MinorID returns the concatenated OCP major and minor version of the branch,
// e.g. "release-4.22" -> "422". This is the prefix of the branch's marketplace
// SKU names and image versions.
func (b Branch) MinorID() (string, error) {
	m := branchNameRE.FindStringSubmatch(b.Name)
	if m == nil {
		return "", fmt.Errorf("branch name %q does not match release-<major>.<minor>", b.Name)
	}
	return m[1] + m[2], nil
}

// Features returns the marketplace plan features of the given architecture.
func (b Branch) Features(arch string) ([]string, error) {
	switch arch {
	case ArchX86_64:
		return b.X86Features, nil
	case ArchAarch64:
		return b.ARMFeatures, nil
	default:
		return nil, fmt.Errorf("unsupported architecture %q", arch)
	}
}
