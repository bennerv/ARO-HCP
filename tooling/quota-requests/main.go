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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Azure/ARO-HCP/tooling/quota-requests/cmd"
)

func main() {
	// Create a root context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := &cobra.Command{
		Use:   "quota-request",
		Short: "Azure quota request management CLI",
		Long: `quota-request is a CLI tool for managing Azure quota requests.

This tool helps automate the process of requesting quota increases for Azure
resources across subscriptions and regions.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	// Add subcommands
	requestCmd, err := cmd.NewRequestCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create request command: %v\n", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(requestCmd)

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
