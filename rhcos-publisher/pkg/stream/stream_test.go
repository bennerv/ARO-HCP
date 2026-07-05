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

package stream

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streamJSON = `{
  "stream": "rhel-9.8",
  "architectures": {
    "x86_64": {
      "artifacts": {
        "azure": {
          "release": "9.8.20260520-0",
          "formats": {
            "vhd.gz": {
              "disk": {
                "location": "https://rhcos.example.com/x86_64.vhd.gz",
                "sha256": "aaa",
                "uncompressed-sha256": "bbb"
              }
            }
          }
        }
      }
    },
    "aarch64": {
      "artifacts": {
        "azure": {
          "release": "9.8.20260520-0",
          "formats": {
            "vhd.gz": {
              "disk": {
                "location": "https://rhcos.example.com/aarch64.vhd.gz",
                "sha256": "ccc",
                "uncompressed-sha256": "ddd"
              }
            }
          }
        }
      }
    },
    "s390x": {
      "artifacts": {
        "metal": {
          "release": "9.8.20260520-0",
          "formats": {}
        }
      }
    }
  }
}`

func TestFetchImages(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		_, _ = w.Write([]byte(streamJSON))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	images, err := client.FetchImages(t.Context(), "release-4.22", 9)
	require.NoError(t, err)
	assert.Equal(t, "/release-4.22/data/data/coreos/coreos-rhel-9.json", requestedPath)

	require.Len(t, images, 2)
	byArch := map[string]Image{}
	for _, image := range images {
		byArch[image.Arch] = image
	}
	assert.Equal(t, Image{
		Branch:             "release-4.22",
		Arch:               "x86_64",
		Release:            "9.8.20260520-0",
		DownloadURL:        "https://rhcos.example.com/x86_64.vhd.gz",
		CompressedSHA256:   "aaa",
		UncompressedSHA256: "bbb",
	}, byArch["x86_64"])
	assert.Equal(t, "release-4.22/x86_64", byArch["x86_64"].Key())
	assert.Contains(t, byArch, "aarch64")
}

func TestFetchImagesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.FetchImages(t.Context(), "release-4.99", 9)
	assert.Error(t, err)
}

func TestFetchImagesIncompleteArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
  "architectures": {
    "x86_64": {
      "artifacts": {
        "azure": {
          "release": "9.8.20260520-0",
          "formats": {
            "vhd.gz": {"disk": {"location": "https://rhcos.example.com/x.vhd.gz", "sha256": "aaa"}}
          }
        }
      }
    }
  }
}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.FetchImages(t.Context(), "release-4.22", 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete azure vhd.gz artifact")
}
