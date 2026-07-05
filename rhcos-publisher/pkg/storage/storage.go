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

// Package storage stages VHDs on the publisher storage account's static
// website. Blobs in the $web container are publicly readable over the web
// endpoint, which is what the marketplace draft references as the image
// source. All access is via Entra ID (shared key access is disabled on the
// account); endpoints are discovered at runtime from the account properties.
package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azcorearm "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

// webContainer is the container behind the storage account static website.
const webContainer = "$web"

// BlobPath is the $web-relative path a staged VHD is uploaded to.
func BlobPath(branch, arch, release string) string {
	return fmt.Sprintf("rhcos/%s/%s/rhcos-%s-azure.%s.vhd", branch, arch, release, arch)
}

// BranchOfBlobPath extracts the branch segment of a path produced by
// BlobPath. It returns false for paths of a different shape.
func BranchOfBlobPath(blobPath string) (string, bool) {
	parts := strings.Split(blobPath, "/")
	if len(parts) != 4 || parts[0] != "rhcos" {
		return "", false
	}
	return parts[1], true
}

// Client stages and purges VHD blobs in the $web container.
type Client struct {
	accounts       *armstorage.AccountsClient
	credential     azcore.TokenCredential
	clientOptions  azcore.ClientOptions
	resourceGroup  string
	accountName    string
	mu             sync.Mutex
	blobEndpoint   string
	webEndpoint    string
	containerCache *container.Client
}

// New builds a storage client for the given account. Endpoints are resolved
// lazily on first use.
func New(subscriptionID, resourceGroup, accountName string, credential azcore.TokenCredential, clientOptions azcore.ClientOptions) (*Client, error) {
	accounts, err := armstorage.NewAccountsClient(subscriptionID, credential, &azcorearm.ClientOptions{ClientOptions: clientOptions})
	if err != nil {
		return nil, fmt.Errorf("failed to create storage accounts client: %w", err)
	}
	return &Client{
		accounts:      accounts,
		credential:    credential,
		clientOptions: clientOptions,
		resourceGroup: resourceGroup,
		accountName:   accountName,
	}, nil
}

// resolve discovers the account's blob and web endpoints and caches a client
// for the $web container.
func (c *Client) resolve(ctx context.Context) (*container.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.containerCache != nil {
		return c.containerCache, nil
	}

	properties, err := c.accounts.GetProperties(ctx, c.resourceGroup, c.accountName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get properties of storage account %s: %w", c.accountName, err)
	}
	if properties.Properties == nil || properties.Properties.PrimaryEndpoints == nil ||
		properties.Properties.PrimaryEndpoints.Blob == nil || properties.Properties.PrimaryEndpoints.Web == nil {
		return nil, fmt.Errorf("storage account %s did not report primary blob/web endpoints", c.accountName)
	}
	c.blobEndpoint = *properties.Properties.PrimaryEndpoints.Blob
	c.webEndpoint = *properties.Properties.PrimaryEndpoints.Web

	serviceClient, err := azblob.NewClient(c.blobEndpoint, c.credential, &azblob.ClientOptions{
		ClientOptions: c.clientOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client for %s: %w", c.blobEndpoint, err)
	}
	c.containerCache = serviceClient.ServiceClient().NewContainerClient(webContainer)
	return c.containerCache, nil
}

// WebURL returns the public static-website URL of a blob path.
func (c *Client) WebURL(ctx context.Context, blobPath string) (string, error) {
	if _, err := c.resolve(ctx); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.TrimSuffix(c.webEndpoint, "/") + "/" + blobPath, nil
}

// BlobExists reports whether the blob exists in the $web container.
func (c *Client) BlobExists(ctx context.Context, blobPath string) (bool, error) {
	containerClient, err := c.resolve(ctx)
	if err != nil {
		return false, err
	}
	_, err = containerClient.NewBlobClient(blobPath).GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check blob %s: %w", blobPath, err)
	}
	return true, nil
}

// UploadFile uploads a local file as a block blob and returns its public
// static-website URL.
func (c *Client) UploadFile(ctx context.Context, blobPath, filePath string) (string, error) {
	containerClient, err := c.resolve(ctx)
	if err != nil {
		return "", err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	blockBlobClient := containerClient.NewBlockBlobClient(blobPath)
	if _, err := blockBlobClient.UploadFile(ctx, file, nil); err != nil {
		return "", fmt.Errorf("failed to upload %s to %s: %w", filePath, blobPath, err)
	}
	return c.WebURL(ctx, blobPath)
}

// DeleteBlob removes a blob; a missing blob is not an error.
func (c *Client) DeleteBlob(ctx context.Context, blobPath string) error {
	containerClient, err := c.resolve(ctx)
	if err != nil {
		return err
	}
	_, err = containerClient.NewBlobClient(blobPath).Delete(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
		return fmt.Errorf("failed to delete blob %s: %w", blobPath, err)
	}
	return nil
}

// ListBlobs returns the paths of all blobs under the given prefix.
func (c *Client) ListBlobs(ctx context.Context, prefix string) ([]string, error) {
	containerClient, err := c.resolve(ctx)
	if err != nil {
		return nil, err
	}
	var paths []string
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{Prefix: &prefix})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if bloberror.HasCode(err, bloberror.ContainerNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list blobs with prefix %s: %w", prefix, err)
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil {
				paths = append(paths, *item.Name)
			}
		}
	}
	return paths, nil
}
