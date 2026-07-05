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

// Package marketplace queries and configures the ARO offering in the Azure
// Marketplace: read access goes through the public VirtualMachineImages ARM
// API, write access through the Partner Center Product Ingestion API.
package marketplace

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azcorearm "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

// VersionLister lists the image versions that exist in the marketplace for a
// SKU. Implementations must treat an unknown SKU as an empty version list.
type VersionLister interface {
	ListVersions(ctx context.Context, publisher, offer, sku string) ([]string, error)
}

// ARMVersionLister queries marketplace image versions through the ARM
// VirtualMachineImages API. Marketplace content is global, but the API is
// addressed per region; any region the offer is available in works.
type ARMVersionLister struct {
	images   *armcompute.VirtualMachineImagesClient
	location string
}

// NewARMVersionLister builds a VersionLister for the given subscription and
// query region.
func NewARMVersionLister(subscriptionID, location string, credential azcore.TokenCredential, clientOptions azcore.ClientOptions) (*ARMVersionLister, error) {
	client, err := armcompute.NewVirtualMachineImagesClient(subscriptionID, credential, &azcorearm.ClientOptions{ClientOptions: clientOptions})
	if err != nil {
		return nil, fmt.Errorf("failed to create VirtualMachineImages client: %w", err)
	}
	return &ARMVersionLister{images: client, location: location}, nil
}

// ListVersions returns the image versions published for the SKU. A SKU that
// does not exist (yet) yields an empty list, not an error.
func (l *ARMVersionLister) ListVersions(ctx context.Context, publisher, offer, sku string) ([]string, error) {
	resp, err := l.images.List(ctx, l.location, publisher, offer, sku, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list marketplace images %s/%s/%s: %w", publisher, offer, sku, err)
	}
	var versions []string
	for _, image := range resp.VirtualMachineImageResourceArray {
		if image != nil && image.Name != nil {
			versions = append(versions, *image.Name)
		}
	}
	return versions, nil
}

var _ VersionLister = (*ARMVersionLister)(nil)
