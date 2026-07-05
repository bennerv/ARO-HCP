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

// Package stream fetches and parses the coreos stream metadata that the
// openshift/installer repository pins per release branch
// (data/data/coreos/coreos-rhel-<N>.json). The stream file is the source of
// truth for RHCOS Azure VHD download URLs, SHA256 hashes and release versions.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

// DefaultBaseURL serves raw files of the openshift/installer repository.
const DefaultBaseURL = "https://raw.githubusercontent.com/openshift/installer"

// maxStreamBytes bounds the stream JSON response size (they are ~100 KiB).
const maxStreamBytes = 16 << 20

// Image describes one downloadable RHCOS Azure VHD of a branch/architecture
// pair, as pinned by the branch's coreos stream file.
type Image struct {
	// Branch is the openshift/installer branch, e.g. "release-4.22".
	Branch string
	// Arch is the coreos architecture name, e.g. "x86_64" or "aarch64".
	Arch string
	// Release is the RHCOS release version, e.g. "9.8.20260520-0".
	Release string
	// DownloadURL is the location of the compressed VHD (.vhd.gz).
	DownloadURL string
	// CompressedSHA256 is the SHA256 of the .vhd.gz artifact.
	CompressedSHA256 string
	// UncompressedSHA256 is the SHA256 of the decompressed .vhd.
	UncompressedSHA256 string
}

// Key is the workqueue key of the image's branch/architecture pair.
func (i Image) Key() string {
	return i.Branch + "/" + i.Arch
}

// Client fetches coreos stream files.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds a stream client. baseURL defaults to DefaultBaseURL when
// empty; tests point it at a local server.
func NewClient(baseURL string) *Client {
	if len(baseURL) == 0 {
		baseURL = DefaultBaseURL
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	return &Client{
		httpClient: retryClient.StandardClient(),
		baseURL:    baseURL,
	}
}

// coreos stream JSON structure, reduced to the fields we consume. See
// github.com/coreos/stream-metadata-go for the full schema.
type streamFile struct {
	Architectures map[string]struct {
		Artifacts map[string]struct {
			Release string `json:"release"`
			Formats map[string]struct {
				Disk *struct {
					Location           string `json:"location"`
					SHA256             string `json:"sha256"`
					UncompressedSHA256 string `json:"uncompressed-sha256"`
				} `json:"disk"`
			} `json:"formats"`
		} `json:"artifacts"`
	} `json:"architectures"`
}

// FetchImages downloads the branch's coreos stream file and returns the Azure
// VHD images it pins, one per architecture that has Azure artifacts.
func (c *Client) FetchImages(ctx context.Context, branch string, rhelVersion int) ([]Image, error) {
	url := fmt.Sprintf("%s/%s/data/data/coreos/coreos-rhel-%d.json", c.baseURL, branch, rhelVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stream file %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stream file %s: HTTP %d", url, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxStreamBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read stream file %s: %w", url, err)
	}

	parsed := &streamFile{}
	if err := json.Unmarshal(raw, parsed); err != nil {
		return nil, fmt.Errorf("failed to parse stream file %s: %w", url, err)
	}

	var images []Image
	for arch, archData := range parsed.Architectures {
		azure, ok := archData.Artifacts["azure"]
		if !ok {
			continue
		}
		format, ok := azure.Formats["vhd.gz"]
		if !ok || format.Disk == nil {
			continue
		}
		if len(azure.Release) == 0 || len(format.Disk.Location) == 0 ||
			len(format.Disk.SHA256) == 0 || len(format.Disk.UncompressedSHA256) == 0 {
			return nil, fmt.Errorf("stream file %s: incomplete azure vhd.gz artifact for %s", url, arch)
		}
		images = append(images, Image{
			Branch:             branch,
			Arch:               arch,
			Release:            azure.Release,
			DownloadURL:        format.Disk.Location,
			CompressedSHA256:   format.Disk.SHA256,
			UncompressedSHA256: format.Disk.UncompressedSHA256,
		})
	}
	return images, nil
}
