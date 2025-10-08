// Copyright 2025 Microsoft Corporation
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

package quota

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	quotaAPIVersion = "2023-02-01"
	baseURL         = "https://management.azure.com"
)

// Client handles interactions with the Azure Quota API
type Client struct {
	credential azcore.TokenCredential
	pipeline   runtime.Pipeline
}

// NewClient creates a new quota API client
func NewClient(subscriptionID, tenantID string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// Create a pipeline for making HTTP requests
	pipeline := runtime.NewPipeline("quota-request", "1.0.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				runtime.NewBearerTokenPolicy(cred, []string{"https://management.azure.com/.default"}, nil),
			},
		},
		nil)

	return &Client{
		credential: cred,
		pipeline:   pipeline,
	}, nil
}

// QuotaRequest represents a quota request payload
type QuotaRequest struct {
	Name       string                 `json:"name"`
	Properties QuotaRequestProperties `json:"properties"`
}

// QuotaRequestProperties represents the properties of a quota request
type QuotaRequestProperties struct {
	Limit          int    `json:"limit"`
	ResourceType   string `json:"resourceType,omitempty"`
	CurrentValue   int    `json:"currentValue,omitempty"`
	Unit           string `json:"unit,omitempty"`
	QuotaPeriod    string `json:"quotaPeriod,omitempty"`
	Properties     any    `json:"properties,omitempty"`
	Name           Name   `json:"name,omitempty"`
	ResourceName   string `json:"resourceName,omitempty"`
	LocalizedValue string `json:"localizedValue,omitempty"`
}

// Name represents the name structure in quota requests
type Name struct {
	Value          string `json:"value,omitempty"`
	LocalizedValue string `json:"localizedValue,omitempty"`
}

// RequestQuota submits a quota increase request
func (c *Client) RequestQuota(ctx context.Context, subscriptionID, region, provider, resourceName string, limit int) error {
	// Construct the quota request URL
	// The format varies by provider, but generally follows this pattern:
	// /subscriptions/{subscriptionID}/providers/{provider}/locations/{region}/quotas/{resourceName}
	url := fmt.Sprintf("%s/subscriptions/%s/providers/%s/locations/%s/quotas/%s?api-version=%s",
		baseURL, subscriptionID, provider, region, resourceName, quotaAPIVersion)

	quotaReq := QuotaRequest{
		Name: resourceName,
		Properties: QuotaRequestProperties{
			Limit: limit,
			Name: Name{
				Value: resourceName,
			},
		},
	}

	body, err := json.Marshal(quotaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal quota request: %w", err)
	}

	req, err := runtime.NewRequest(ctx, http.MethodPut, url)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := req.SetBody(streaming.NopCloser(bytes.NewReader(body)), "application/json"); err != nil {
		return fmt.Errorf("failed to set request body: %w", err)
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("quota request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetCurrentQuota retrieves the current quota for a resource
func (c *Client) GetCurrentQuota(ctx context.Context, subscriptionID, region, provider, resourceName string) (*QuotaRequestProperties, error) {
	url := fmt.Sprintf("%s/subscriptions/%s/providers/%s/locations/%s/quotas/%s?api-version=%s",
		baseURL, subscriptionID, provider, region, resourceName, quotaAPIVersion)

	req, err := runtime.NewRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get quota failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var quotaResp QuotaRequest
	if err := json.NewDecoder(resp.Body).Decode(&quotaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &quotaResp.Properties, nil
}
