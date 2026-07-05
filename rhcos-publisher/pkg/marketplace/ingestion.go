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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// Product Ingestion API constants. See
// https://learn.microsoft.com/en-us/partner-center/marketplace/product-ingestion-api
const (
	// DefaultIngestionBaseURL is the Product Ingestion API endpoint.
	DefaultIngestionBaseURL = "https://graph.microsoft.com/rp/product-ingestion"
	// ingestionAPIVersion is the $version query parameter value.
	ingestionAPIVersion = "2022-03-01-preview3"
	// ingestionTokenScope is the OAuth2 scope of the API.
	ingestionTokenScope = "https://graph.microsoft.com/.default"

	configureSchema  = "https://schema.mp.microsoft.com/schema/configure/2022-03-01-preview3"
	planSchema       = "https://schema.mp.microsoft.com/schema/plan/2022-03-01-preview3"
	techConfigSchema = "https://schema.mp.microsoft.com/schema/virtual-machine-plan-technical-configuration/2022-03-01-preview3"

	// techConfigSchemaPrefix identifies VM plan technical configuration
	// resources in a resource tree, regardless of schema revision.
	techConfigSchemaPrefix = "https://schema.mp.microsoft.com/schema/virtual-machine-plan-technical-configuration/"

	maxIngestionResponseBytes = 64 << 20
)

// Job states of the configure endpoint.
const (
	jobStatusCompleted = "completed"
	jobResultSucceeded = "succeeded"
)

// IngestionClient is the subset of the Product Ingestion API used by the
// publisher controller. All resources are handled as raw JSON objects
// (map[string]any) so that fields we do not model are preserved when a
// fetched resource is modified and posted back.
type IngestionClient interface {
	// GetProductIDByExternalID resolves an offer external ID (e.g. "aro4") to
	// the product durable ID ("product/<guid>").
	GetProductIDByExternalID(ctx context.Context, publisher, externalID string) (string, error)
	// GetResourceTree fetches the draft resource tree of a product.
	GetResourceTree(ctx context.Context, productDurableID string) (*ResourceTree, error)
	// Configure submits resources to the configure endpoint and returns the
	// job ID.
	Configure(ctx context.Context, resources []map[string]any) (string, error)
	// WaitForJob polls the configure job until it finishes and returns an
	// error if it did not succeed.
	WaitForJob(ctx context.Context, jobID string) error
}

// ResourceTree is a product's draft resource tree.
type ResourceTree struct {
	Root      string           `json:"root"`
	Resources []map[string]any `json:"resources"`
}

// HTTPIngestionClient talks to the live Product Ingestion API, authenticating
// with a bearer token from the given credential.
type HTTPIngestionClient struct {
	httpClient *http.Client
	credential azcore.TokenCredential
	baseURL    string
}

// NewHTTPIngestionClient builds an ingestion client. baseURL defaults to
// DefaultIngestionBaseURL when empty.
func NewHTTPIngestionClient(credential azcore.TokenCredential, baseURL string) *HTTPIngestionClient {
	if len(baseURL) == 0 {
		baseURL = DefaultIngestionBaseURL
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	return &HTTPIngestionClient{
		httpClient: retryClient.StandardClient(),
		credential: credential,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}
}

var _ IngestionClient = (*HTTPIngestionClient)(nil)

func (c *HTTPIngestionClient) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{ingestionTokenScope}})
	if err != nil {
		return fmt.Errorf("failed to acquire ingestion API token: %w", err)
	}

	if query == nil {
		query = url.Values{}
	}
	query.Set("$version", ingestionAPIVersion)

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path+"?"+query.Encode(), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ingestion API %s %s failed: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxIngestionResponseBytes))
	if err != nil {
		return fmt.Errorf("ingestion API %s %s: failed to read response: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("ingestion API %s %s: HTTP %d: %s", method, path, resp.StatusCode, string(raw))
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("ingestion API %s %s: failed to parse response: %w", method, path, err)
		}
	}
	return nil
}

// GetProductIDByExternalID pages through /product and returns the durable ID
// of the product whose identity.externalId matches.
func (c *HTTPIngestionClient) GetProductIDByExternalID(ctx context.Context, publisher, externalID string) (string, error) {
	path := "/product"
	for {
		var page struct {
			Value []struct {
				ID       string `json:"id"`
				Identity struct {
					ExternalID string `json:"externalId"`
				} `json:"identity"`
			} `json:"value"`
			NextLink string `json:"@nextLink"`
		}
		if err := c.do(ctx, http.MethodGet, path, nil, nil, &page); err != nil {
			return "", err
		}
		for _, product := range page.Value {
			if product.Identity.ExternalID == externalID {
				return product.ID, nil
			}
		}
		if len(page.NextLink) == 0 {
			return "", fmt.Errorf("no product with external ID %q found for publisher %q", externalID, publisher)
		}
		if strings.HasPrefix(page.NextLink, "http://") || strings.HasPrefix(page.NextLink, "https://") {
			parsed, err := url.Parse(page.NextLink)
			if err != nil {
				return "", fmt.Errorf("failed to parse @nextLink %q: %w", page.NextLink, err)
			}
			path = parsed.Path
		} else {
			path = "/" + strings.TrimPrefix(page.NextLink, "/")
		}
	}
}

// GetResourceTree fetches the product's draft resource tree.
func (c *HTTPIngestionClient) GetResourceTree(ctx context.Context, productDurableID string) (*ResourceTree, error) {
	tree := &ResourceTree{}
	query := url.Values{"targetType": []string{"draft"}}
	if err := c.do(ctx, http.MethodGet, "/resource-tree/"+productDurableID, query, nil, tree); err != nil {
		return nil, err
	}
	return tree, nil
}

// Configure submits resources for asynchronous processing.
func (c *HTTPIngestionClient) Configure(ctx context.Context, resources []map[string]any) (string, error) {
	body := map[string]any{
		"$schema":   configureSchema,
		"resources": resources,
	}
	var result struct {
		JobID string `json:"jobId"`
	}
	if err := c.do(ctx, http.MethodPost, "/configure", nil, body, &result); err != nil {
		return "", err
	}
	if len(result.JobID) == 0 {
		return "", fmt.Errorf("configure response did not contain a job ID")
	}
	return result.JobID, nil
}

// WaitForJob polls the configure job status until it completes, the context
// is cancelled, or the job fails.
func (c *HTTPIngestionClient) WaitForJob(ctx context.Context, jobID string) error {
	return wait.PollUntilContextCancel(ctx, 15*time.Second, true, func(ctx context.Context) (bool, error) {
		var status struct {
			JobStatus string `json:"jobStatus"`
			JobResult string `json:"jobResult"`
			Errors    []any  `json:"errors"`
		}
		if err := c.do(ctx, http.MethodGet, "/configure/"+jobID+"/status", nil, nil, &status); err != nil {
			return false, err
		}
		if status.JobStatus != jobStatusCompleted {
			return false, nil
		}
		if status.JobResult != jobResultSucceeded {
			return false, fmt.Errorf("configure job %s finished with result %q: %v", jobID, status.JobResult, status.Errors)
		}
		return true, nil
	})
}

//
// Resource tree helpers. These operate on raw JSON objects so unknown fields
// survive a fetch-modify-configure round trip.
//

func resourceString(resource map[string]any, key string) string {
	value, _ := resource[key].(string)
	return value
}

// FindPlanDurableID locates a plan resource by its external ID (SKU name) and
// returns its durable ID ("plan/<product-guid>/<plan-guid>").
func FindPlanDurableID(tree *ResourceTree, externalID string) (string, bool) {
	for _, resource := range tree.Resources {
		id := resourceString(resource, "id")
		if !strings.HasPrefix(id, "plan/") {
			continue
		}
		identity, _ := resource["identity"].(map[string]any)
		if identity != nil && identity["externalId"] == externalID {
			return id, true
		}
	}
	return "", false
}

// FindTechConfig locates the VM technical configuration resource of a plan.
func FindTechConfig(tree *ResourceTree, planDurableID string) (map[string]any, bool) {
	for _, resource := range tree.Resources {
		schema := resourceString(resource, "$schema")
		if !strings.HasPrefix(schema, techConfigSchemaPrefix) {
			continue
		}
		if resourceString(resource, "plan") == planDurableID {
			return resource, true
		}
	}
	return nil, false
}

// NewPlanResource builds a plan creation resource for the configure endpoint.
func NewPlanResource(productDurableID, externalID, displayName string) map[string]any {
	return map[string]any{
		"$schema": planSchema,
		"product": productDurableID,
		"identity": map[string]any{
			"externalId": externalID,
		},
		"alias":        displayName,
		"azureRegions": []any{"azureGlobal"},
	}
}

// NewTechConfigResource builds a minimal VM technical configuration for a
// plan that does not have one yet (freshly created plans).
func NewTechConfigResource(productDurableID, planDurableID string) map[string]any {
	return map[string]any{
		"$schema": techConfigSchema,
		"product": productDurableID,
		"plan":    planDurableID,
		"operatingSystem": map[string]any{
			"family":       "linux",
			"friendlyName": "Red Hat Enterprise Linux CoreOS",
		},
	}
}

// EnsureSKUs adds any missing SKU entries to the technical configuration and
// reports whether it modified the resource. Existing SKU entries are left
// untouched.
func EnsureSKUs(techConfig map[string]any, skus []SKU) bool {
	existing, _ := techConfig["skus"].([]any)
	present := map[string]bool{}
	for _, entry := range existing {
		sku, _ := entry.(map[string]any)
		if sku != nil {
			present[resourceString(sku, "skuId")] = true
		}
	}

	changed := false
	for _, sku := range skus {
		if present[sku.Name] {
			continue
		}
		entry := map[string]any{
			"skuId":     sku.Name,
			"imageType": sku.ImageType,
		}
		if len(sku.SecurityTypes) > 0 {
			securityTypes := make([]any, 0, len(sku.SecurityTypes))
			for _, securityType := range sku.SecurityTypes {
				securityTypes = append(securityTypes, securityType)
			}
			entry["securityType"] = securityTypes
		}
		existing = append(existing, entry)
		changed = true
	}
	if changed {
		techConfig["skus"] = existing
	}
	return changed
}

// VMImage is one image definition of a marketplace image version.
type VMImage struct {
	// ImageType is the Product Ingestion API image type, e.g. "x64Gen2".
	ImageType string
	// URI is the publicly downloadable VHD location.
	URI string
}

// EnsureImageVersion adds the image version to the technical configuration if
// it is not present yet and reports whether it modified the resource. An
// existing version with the same number is never modified.
func EnsureImageVersion(techConfig map[string]any, version string, images []VMImage) bool {
	existing, _ := techConfig["vmImageVersions"].([]any)
	for _, entry := range existing {
		imageVersion, _ := entry.(map[string]any)
		if imageVersion != nil && resourceString(imageVersion, "versionNumber") == version {
			return false
		}
	}

	vmImages := make([]any, 0, len(images))
	for _, image := range images {
		vmImages = append(vmImages, map[string]any{
			"imageType": image.ImageType,
			"source": map[string]any{
				"sourceType": "sasUri",
				"osDisk": map[string]any{
					"uri": image.URI,
				},
				"dataDisks": []any{},
			},
		})
	}
	techConfig["vmImageVersions"] = append(existing, map[string]any{
		"versionNumber": version,
		"vmImages":      vmImages,
	})
	return true
}
