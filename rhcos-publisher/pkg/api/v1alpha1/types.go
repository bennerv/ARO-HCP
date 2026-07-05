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

// Package v1alpha1 contains API types for the rhcos.aro.openshift.io API group.
// An RHCOSRelease tracks the publishing state of the RHCOS Azure images of one
// OCP release branch. The resources are created by the rhcos-publisher Helm
// chart (one per configured branch); the controller only ever writes the
// status subresource.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImagePhase describes where an RHCOS image of a branch/architecture pair is
// in the publishing lifecycle.
type ImagePhase string

const (
	// ImagePhaseStaged means the VHD has been uploaded to the staging storage
	// account's static website and is publicly downloadable, but no marketplace
	// draft has been configured yet.
	ImagePhaseStaged ImagePhase = "staged"
	// ImagePhaseDraft means the marketplace plan and image version have been
	// configured as a draft in Partner Center and await a manual publish.
	ImagePhaseDraft ImagePhase = "draft"
	// ImagePhasePublished means the image version is live in the Azure
	// Marketplace and the staged VHD has been purged.
	ImagePhasePublished ImagePhase = "published"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHCOSRelease tracks the RHCOS Azure images of one OCP release branch and
// their progress towards the ARO 1P marketplace.
type RHCOSRelease struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec identifies the release branch and the marketplace features of its
	// images. It is rendered from the ARO-HCP configuration by the Helm chart
	// and never modified by the controller.
	Spec RHCOSReleaseSpec `json:"spec"`

	// +optional
	// status contains the per-architecture publishing state observed by the
	// controller.
	Status RHCOSReleaseStatus `json:"status,omitempty,omitzero"`
}

// RHCOSReleaseSpec identifies an OCP release branch to mirror.
type RHCOSReleaseSpec struct {
	// branch is the openshift/installer branch to poll for coreos stream
	// metadata, e.g. "release-4.22".
	Branch string `json:"branch"`

	// rhelVersion is the major RHEL version of the coreos stream used by the
	// branch (e.g. 9 selects coreos-rhel-9.json).
	RHELVersion int `json:"rhelVersion"`

	// +optional
	// armFeatures are the marketplace plan features of the aarch64 images.
	ARMFeatures []string `json:"armFeatures,omitempty"`

	// +optional
	// x86Features are the marketplace plan features of the x86_64 images.
	X86Features []string `json:"x86Features,omitempty"`
}

// RHCOSReleaseStatus is the observed publishing state of a branch.
type RHCOSReleaseStatus struct {
	// +optional
	// architectures maps a coreos architecture name ("x86_64", "aarch64") to
	// the publishing state of its current image.
	Architectures map[string]RHCOSReleaseArchStatus `json:"architectures,omitempty"`

	// +optional
	// lastInstallerSync is when the installer lister last successfully fetched
	// the branch's coreos stream metadata.
	LastInstallerSync *metav1.Time `json:"lastInstallerSync,omitempty"`

	// +optional
	// lastMarketplaceSync is when the marketplace lister last successfully
	// queried the branch's marketplace SKUs.
	LastMarketplaceSync *metav1.Time `json:"lastMarketplaceSync,omitempty"`
}

// RHCOSReleaseArchStatus is the publishing state of one architecture's image.
type RHCOSReleaseArchStatus struct {
	// +optional
	// release is the RHCOS release version currently being tracked, e.g.
	// "9.8.20260520-0".
	Release string `json:"release,omitempty"`

	// +optional
	// stagedURL is the public static-website URL of the staged VHD. Cleared
	// once the image is published and the VHD purged.
	StagedURL string `json:"stagedURL,omitempty"`

	// +optional
	// phase is the publishing lifecycle phase: staged, draft or published.
	Phase ImagePhase `json:"phase,omitempty"`

	// +optional
	// configureJobID is the Partner Center Product Ingestion API job that
	// configured the marketplace draft for this image.
	ConfigureJobID string `json:"configureJobId,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHCOSReleaseList is a list of RHCOSRelease resources.
type RHCOSReleaseList struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ListMeta `json:"metadata,omitempty,omitzero"`

	Items []RHCOSRelease `json:"items"`
}
