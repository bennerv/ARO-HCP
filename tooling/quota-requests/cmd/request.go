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

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Azure/ARO-HCP/tooling/quota-requests/pkg/config"
	"github.com/Azure/ARO-HCP/tooling/quota-requests/pkg/quota"
)

// RequestOptions contains the options for the request command
type RequestOptions struct {
	SubscriptionID string
	TenantID       string
	Region         string
	ConfigPath     string
}

// NewRequestCommand creates the request command
func NewRequestCommand() (*cobra.Command, error) {
	opts := &RequestOptions{}

	cmd := &cobra.Command{
		Use:   "request",
		Short: "Request quota increases for Azure resources",
		Long: `Request quota increases for Azure resources based on a configuration file.

This command reads a configuration file containing quota requests and submits
them to the Azure Quota API for the specified subscription and region.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRequest(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.SubscriptionID, "subscription-id", "s", "", "Azure subscription ID")
	cmd.Flags().StringVarP(&opts.Region, "region", "r", "", "Azure region (e.g., eastus, westus2)")
	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "quota-config.yaml", "Path to quota configuration file")
	cmd.Flags().StringVarP(&opts.TenantID, "tenant-id", "t", "", "Azure tenant ID (optional, uses default if not specified)")

	if err := cmd.MarkFlagRequired("subscription-id"); err != nil {
		return nil, fmt.Errorf("failed to mark subscription-id flag as required: %w", err)
	}

	if err := cmd.MarkFlagRequired("region"); err != nil {
		return nil, fmt.Errorf("failed to mark region flag as required: %w", err)
	}

	if err := cmd.MarkFlagFilename("config", "yaml", "yml"); err != nil {
		return nil, fmt.Errorf("failed to mark config flag as filename: %w", err)
	}

	return cmd, nil
}

func runRequest(ctx context.Context, opts *RequestOptions) error {
	// Load configuration
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Loaded %d quota request(s) from %s\n", len(cfg.Quotas), opts.ConfigPath)

	// Create quota client
	client, err := quota.NewClient(opts.SubscriptionID, opts.TenantID)
	if err != nil {
		return fmt.Errorf("failed to create quota client: %w", err)
	}

	fmt.Printf("Requesting quotas for subscription %s in region %s\n", opts.SubscriptionID, opts.Region)

	// Submit each quota request
	for i, quotaReq := range cfg.Quotas {
		fmt.Printf("[%d/%d] Requesting quota for %s/%s (limit: %d)...\n",
			i+1, len(cfg.Quotas), quotaReq.Provider, quotaReq.ResourceName, quotaReq.Limit)

		err := client.RequestQuota(ctx, opts.SubscriptionID, opts.Region, quotaReq.Provider, quotaReq.ResourceName, quotaReq.Limit)
		if err != nil {
			fmt.Printf("  ✗ Failed: %v\n", err)
			continue
		}
		fmt.Printf("  ✓ Success\n")
	}

	return nil
}
