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

package readonly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/go-logr/logr"
	"github.com/microsoft/go-otel-audit/audit/base"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/spf13/cobra"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/Azure/ARO-HCP/admin/server/server"
	"github.com/Azure/ARO-HCP/internal/audit"
	"github.com/Azure/ARO-HCP/internal/azsdk"
	"github.com/Azure/ARO-HCP/internal/database"
	"github.com/Azure/ARO-HCP/internal/utils"
)

func DefaultOptions() *RawOptions {
	return &RawOptions{
		Port:               8443,
		MetricsPort:        8444,
		AuditLogQueueSize:  2048,
		CosmosURL:          os.Getenv("COSMOS_URL"),
		CosmosName:         os.Getenv("COSMOS_NAME"),
		AuditConnectSocket: os.Getenv("AUDIT_CONNECT_SOCKET") == "true",
	}
}

type RawOptions struct {
	LogVerbosity       int
	Port               int
	MetricsPort        int
	Location           string
	CosmosURL          string
	CosmosName         string
	AuditLogQueueSize  int
	AuditConnectSocket bool
}

func (opts *RawOptions) BindOptions(cmd *cobra.Command) error {
	cmd.Flags().IntVar(&opts.Port, "port", opts.Port, "Port to serve content on.")
	cmd.Flags().IntVar(&opts.MetricsPort, "metrics-port", opts.MetricsPort, "Port to serve metrics on.")
	cmd.Flags().StringVar(&opts.Location, "location", opts.Location, "Location to serve content on.")
	cmd.Flags().StringVar(&opts.CosmosURL, "cosmos-url", opts.CosmosURL, "URL of the Cosmos DB.")
	cmd.Flags().StringVar(&opts.CosmosName, "cosmos-name", opts.CosmosName, "Name of the Cosmos DB.")
	cmd.Flags().IntVar(&opts.AuditLogQueueSize, "audit-log-queue-size", opts.AuditLogQueueSize, "Log queue size for audit logging client.")
	cmd.Flags().BoolVar(&opts.AuditConnectSocket, "audit-connect-socket", opts.AuditConnectSocket, "Connect to mdsd audit socket.")
	return nil
}

type validatedOptions struct {
	*RawOptions
}

type ValidatedOptions struct {
	*validatedOptions
}

type completedOptions struct {
	Port                 int
	MetricsPort          int
	Location             string
	CosmosDatabaseClient *azcosmos.DatabaseClient
	ResourcesDBClient    database.ResourcesDBClient
	BillingDBClient      database.BillingDBClient
	AuditClient          audit.Client
	Registry             *prometheus.Registry
}

type Options struct {
	*completedOptions
}

func (o *RawOptions) Validate() (*ValidatedOptions, error) {
	if o.Location == "" {
		return nil, fmt.Errorf("location is required")
	}
	if o.CosmosURL == "" {
		return nil, fmt.Errorf("cosmos-url is required")
	}
	if o.CosmosName == "" {
		return nil, fmt.Errorf("cosmos-name is required")
	}
	return &ValidatedOptions{
		validatedOptions: &validatedOptions{
			RawOptions: o,
		},
	}, nil
}

func (o *ValidatedOptions) Complete(ctx context.Context) (*Options, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	clientOpts := azsdk.NewClientOptions(azsdk.ComponentAdmin)
	// FIXME Cloud should be determined by other means.
	clientOpts.Cloud = cloud.AzurePublic
	cosmosDatabaseClient, err := database.NewCosmosDatabaseClient(
		o.CosmosURL,
		o.CosmosName,
		clientOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the CosmosDB client: %w", err)
	}
	resourcesDBClient, err := database.NewResourcesDBClient(cosmosDatabaseClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create the resources DB client: %w", err)
	}
	billingDBClient, err := database.NewBillingDBClient(cosmosDatabaseClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create the billing database client: %w", err)
	}

	logger := utils.LoggerFromContext(ctx)
	slogLogger := slog.New(logr.ToSlogHandler(logger))
	auditClient, err := audit.NewOtelAuditClient(
		ctx,
		audit.CreateConn(o.AuditConnectSocket),
		registry,
		base.WithLogger(slogLogger),
		base.WithSettings(base.Settings{
			QueueSize: o.AuditLogQueueSize,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit client: %w", err)
	}

	return &Options{
		completedOptions: &completedOptions{
			Port:                 o.Port,
			MetricsPort:          o.MetricsPort,
			Location:             o.Location,
			CosmosDatabaseClient: cosmosDatabaseClient,
			ResourcesDBClient:    resourcesDBClient,
			BillingDBClient:      billingDBClient,
			AuditClient:          auditClient,
			Registry:             registry,
		},
	}, nil
}

func (opts *Options) Run(ctx context.Context) error {
	logger := utils.LoggerFromContext(ctx)

	listener, err := net.Listen("tcp", net.JoinHostPort("", fmt.Sprintf("%d", opts.Port)))
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	metricsListener, err := net.Listen("tcp", net.JoinHostPort("", fmt.Sprintf("%d", opts.MetricsPort)))
	if err != nil {
		return fmt.Errorf("failed to create metrics listener: %w", err)
	}

	adminAPI := server.NewReadOnlyAdminAPI(
		logger,
		opts.Location,
		listener,
		metricsListener,
		opts.CosmosDatabaseClient,
		opts.ResourcesDBClient,
		opts.BillingDBClient,
		opts.AuditClient,
		opts.Registry,
	)

	runErrCh := make(chan error, 1)
	go func() {
		defer utilruntime.HandleCrash()
		runErrCh <- adminAPI.Run(ctx)
		logger.Info("admin api (read-only) exited")
	}()

	<-ctx.Done()
	logger.Info("context closed")

	logger.Info("waiting for run to finish")
	runErr := <-runErrCh
	return runErr
}
