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

// Package download fetches compressed RHCOS VHDs, verifying the SHA256 of
// both the compressed and the decompressed artifact against the hashes pinned
// in the coreos stream metadata.
package download

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

// Downloader downloads and verifies RHCOS VHDs into a working directory.
// VHDs decompress to ~16 GiB, so workDir must be backed by a volume with
// enough headroom for one compressed plus one decompressed image.
type Downloader struct {
	httpClient *http.Client
	workDir    string
}

// New builds a Downloader that stages files under workDir.
func New(workDir string) *Downloader {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	return &Downloader{
		httpClient: retryClient.StandardClient(),
		workDir:    workDir,
	}
}

var knownArchitectures = map[string]bool{
	"x86_64":  true,
	"aarch64": true,
}

// FetchVHD downloads the image's compressed VHD, verifies its SHA256,
// decompresses it, and verifies the SHA256 of the decompressed VHD. It
// returns the path of the verified .vhd file; the compressed intermediate is
// deleted before return. The caller is responsible for deleting the returned
// file. On error, all temporary files are cleaned up.
func (d *Downloader) FetchVHD(ctx context.Context, image stream.Image) (_ string, err error) {
	if !knownArchitectures[image.Arch] {
		return "", fmt.Errorf("unsupported architecture %q", image.Arch)
	}
	compressedPath := filepath.Join(d.workDir, fmt.Sprintf("rhcos-%s-azure.%s.vhd.gz", image.Release, image.Arch))
	vhdPath := filepath.Join(d.workDir, fmt.Sprintf("rhcos-%s-azure.%s.vhd", image.Release, image.Arch))
	defer func() {
		_ = os.Remove(compressedPath)
		if err != nil {
			_ = os.Remove(vhdPath)
		}
	}()

	if err := d.downloadFile(ctx, image.DownloadURL, compressedPath, image.CompressedSHA256); err != nil {
		return "", err
	}
	if err := decompressFile(compressedPath, vhdPath, image.UncompressedSHA256); err != nil {
		return "", err
	}
	return vhdPath, nil
}

// downloadFile streams url into path, verifying the SHA256 of the content.
func (d *Downloader) downloadFile(ctx context.Context, url, path, expectedSHA256 string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", url, resp.StatusCode)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(file, io.TeeReader(resp.Body, hasher))
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write %s: %w", path, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close %s: %w", path, closeErr)
	}

	if actual := fmt.Sprintf("%x", hasher.Sum(nil)); actual != expectedSHA256 {
		return fmt.Errorf("SHA256 mismatch for %s: expected %s, got %s", url, expectedSHA256, actual)
	}
	return nil
}

// decompressFile gunzips src into dst, verifying the SHA256 of the
// decompressed content.
func decompressFile(src, dst, expectedSHA256 string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decompress %s: %w", src, err)
	}
	defer func() { _ = gzReader.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(dstFile, io.TeeReader(gzReader, hasher)) //nolint:gosec // trusted source, output verified by SHA256 below
	closeErr := dstFile.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to decompress %s: %w", src, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close %s: %w", dst, closeErr)
	}

	if actual := fmt.Sprintf("%x", hasher.Sum(nil)); actual != expectedSHA256 {
		return fmt.Errorf("SHA256 mismatch for decompressed %s: expected %s, got %s", src, expectedSHA256, actual)
	}
	return nil
}
