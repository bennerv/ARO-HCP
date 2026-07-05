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
	"fmt"
	"regexp"
	"slices"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
)

// Marketplace image types of the Product Ingestion API.
const (
	ImageTypeX64Gen2   = "x64Gen2"
	ImageTypeARM64Gen2 = "arm64Gen2"
)

// releaseRE parses RHCOS release versions like "9.8.20260520-0" into the RHEL
// major/minor version and the build date.
var releaseRE = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d{8})\d*-\d+$`)

// SKU describes one marketplace SKU (plan) of a branch/architecture pair.
type SKU struct {
	// Name is the SKU / plan external ID, e.g. "aro_4-22_x64_gen2".
	Name string
	// ImageType is the Product Ingestion API image type, e.g. "x64Gen2".
	ImageType string
	// SecurityTypes are the security capabilities of the SKU, e.g. ["trusted"].
	SecurityTypes []string
}

// SKUsForArch derives the marketplace SKUs of a branch/architecture pair
// following the established ARO convention:
//
//	x86_64  Gen2: aro_<major>-<minor>_x64_gen2
//	aarch64 Gen2: aro_<major>-<minor>_arm_gen2
func SKUsForArch(branch config.Branch, arch string) ([]SKU, error) {
	versionSlug, err := branch.VersionSlug()
	if err != nil {
		return nil, err
	}
	features, err := branch.Features(arch)
	if err != nil {
		return nil, err
	}

	var securityTypes []string
	if slices.Contains(features, config.FeatureTrustedLaunch) {
		securityTypes = []string{"trusted"}
	}

	switch arch {
	case config.ArchX86_64:
		return []SKU{{
			Name:          fmt.Sprintf("aro_%s_x64_gen2", versionSlug),
			ImageType:     ImageTypeX64Gen2,
			SecurityTypes: securityTypes,
		}}, nil
	case config.ArchAarch64:
		return []SKU{{
			Name:          fmt.Sprintf("aro_%s_arm_gen2", versionSlug),
			ImageType:     ImageTypeARM64Gen2,
			SecurityTypes: securityTypes,
		}}, nil
	default:
		return nil, fmt.Errorf("unsupported architecture %q", arch)
	}
}

// ImageVersion derives the marketplace image version of an RHCOS release
// following the established ARO convention
// <ocpMinorID>.<rhelMajorMinor>.<buildDate>, e.g. branch minor "422" and
// release "9.8.20260520-0" become "422.98.20260520". Azure requires versions
// to be numeric Major.Minor.Patch triples.
func ImageVersion(branchMinorID, release string) (string, error) {
	m := releaseRE.FindStringSubmatch(release)
	if m == nil {
		return "", fmt.Errorf("release %q does not match <rhelMajor>.<rhelMinor>.<date>-<build>", release)
	}
	return fmt.Sprintf("%s.%s%s.%s", branchMinorID, m[1], m[2], m[3]), nil
}
