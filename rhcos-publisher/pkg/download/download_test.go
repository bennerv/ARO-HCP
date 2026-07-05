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

package download

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

// gzipPayload compresses content and returns (compressed, compressedSHA, uncompressedSHA).
func gzipPayload(t *testing.T, content []byte) ([]byte, string, string) {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, gzWriter.Close())
	return buf.Bytes(),
		fmt.Sprintf("%x", sha256.Sum256(buf.Bytes())),
		fmt.Sprintf("%x", sha256.Sum256(content))
}

func TestFetchVHD(t *testing.T) {
	content := []byte("pretend this is a VHD")
	compressed, compressedSHA, uncompressedSHA := gzipPayload(t, content)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(compressed)
	}))
	defer server.Close()

	workDir := t.TempDir()
	downloader := New(workDir)
	vhdPath, err := downloader.FetchVHD(t.Context(), stream.Image{
		Branch:             "release-4.22",
		Arch:               "x86_64",
		Release:            "9.8.20260520-0",
		DownloadURL:        server.URL,
		CompressedSHA256:   compressedSHA,
		UncompressedSHA256: uncompressedSHA,
	})
	require.NoError(t, err)

	got, err := os.ReadFile(vhdPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	// The compressed intermediate must be gone.
	entries, err := os.ReadDir(workDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, filepath.Base(vhdPath), entries[0].Name())
}

func TestFetchVHDCompressedHashMismatch(t *testing.T) {
	compressed, _, uncompressedSHA := gzipPayload(t, []byte("payload"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(compressed)
	}))
	defer server.Close()

	workDir := t.TempDir()
	downloader := New(workDir)
	_, err := downloader.FetchVHD(t.Context(), stream.Image{
		Release:            "9.8.20260520-0",
		Arch:               "x86_64",
		DownloadURL:        server.URL,
		CompressedSHA256:   "wrong",
		UncompressedSHA256: uncompressedSHA,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SHA256 mismatch")

	entries, err := os.ReadDir(workDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "temporary files must be cleaned up on error")
}

func TestFetchVHDUncompressedHashMismatch(t *testing.T) {
	compressed, compressedSHA, _ := gzipPayload(t, []byte("payload"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(compressed)
	}))
	defer server.Close()

	workDir := t.TempDir()
	downloader := New(workDir)
	_, err := downloader.FetchVHD(t.Context(), stream.Image{
		Release:            "9.8.20260520-0",
		Arch:               "x86_64",
		DownloadURL:        server.URL,
		CompressedSHA256:   compressedSHA,
		UncompressedSHA256: "wrong",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SHA256 mismatch for decompressed")

	entries, err := os.ReadDir(workDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "temporary files must be cleaned up on error")
}
