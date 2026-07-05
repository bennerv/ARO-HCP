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

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/Azure/ARO-HCP/internal/azsdk"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/download"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/manager"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/status"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/storage"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

const (
	defaultHealthzListenAddress = ":8080"
	defaultMetricsListenAddress = ":8081"
	defaultLeaderElectionID     = "rhcos-publisher-controller"
	defaultMarketplaceLocation  = "centralus"
	defaultInstallerCooldown    = time.Hour
	defaultMarketplaceCooldown  = 2 * time.Hour
	defaultWorkDir              = "/tmp"
)

type RawControllerOptions struct {
	ConfigPath string

	SubscriptionID     string
	ResourceGroup      string
	StorageAccountName string

	CloudEnvironment    string
	MarketplaceLocation string

	InstallerCooldown         time.Duration
	MarketplaceCooldown       time.Duration
	MarketplacePublishEnabled bool
	WorkDir                   string

	KubeNamespace        string
	LeaderElectionID     string
	HealthzListenAddress string
	MetricsListenAddress string
}

func DefaultControllerOptions() *RawControllerOptions {
	return &RawControllerOptions{
		CloudEnvironment:     "AzurePublicCloud",
		MarketplaceLocation:  defaultMarketplaceLocation,
		InstallerCooldown:    defaultInstallerCooldown,
		MarketplaceCooldown:  defaultMarketplaceCooldown,
		WorkDir:              defaultWorkDir,
		HealthzListenAddress: defaultHealthzListenAddress,
		MetricsListenAddress: defaultMetricsListenAddress,
		LeaderElectionID:     defaultLeaderElectionID,
	}
}

func BindControllerOptions(opts *RawControllerOptions, cmd *cobra.Command) error {
	cmd.Flags().StringVar(&opts.ConfigPath, "config", opts.ConfigPath, "path to the branch/marketplace configuration file")
	cmd.Flags().StringVar(&opts.SubscriptionID, "subscription-id", opts.SubscriptionID, "Azure subscription of the staging storage account")
	cmd.Flags().StringVar(&opts.ResourceGroup, "resource-group", opts.ResourceGroup, "resource group of the staging storage account")
	cmd.Flags().StringVar(&opts.StorageAccountName, "storage-account-name", opts.StorageAccountName, "name of the staging storage account")
	cmd.Flags().StringVar(&opts.CloudEnvironment, "cloud-environment", opts.CloudEnvironment, "Azure cloud environment (AzurePublicCloud, AzureChinaCloud, AzureUSGovernmentCloud)")
	cmd.Flags().StringVar(&opts.MarketplaceLocation, "marketplace-location", opts.MarketplaceLocation, "Azure region used to query marketplace image versions")
	cmd.Flags().DurationVar(&opts.InstallerCooldown, "installer-cooldown", opts.InstallerCooldown, "interval between coreos stream metadata polls")
	cmd.Flags().DurationVar(&opts.MarketplaceCooldown, "marketplace-cooldown", opts.MarketplaceCooldown, "interval between marketplace image version polls")
	cmd.Flags().BoolVar(&opts.MarketplacePublishEnabled, "marketplace-publish-enabled", opts.MarketplacePublishEnabled, "configure marketplace drafts via the Partner Center Product Ingestion API (requires marketplace account access)")
	cmd.Flags().StringVar(&opts.WorkDir, "work-dir", opts.WorkDir, "scratch directory for VHD downloads (needs ~20GiB headroom)")
	cmd.Flags().StringVar(&opts.KubeNamespace, "kube-namespace", opts.KubeNamespace, "Kubernetes namespace of the leader election lease and the RHCOSRelease resources")
	cmd.Flags().StringVar(&opts.LeaderElectionID, "leader-election-id", opts.LeaderElectionID, "name of the leader election lease")
	cmd.Flags().StringVar(&opts.HealthzListenAddress, "healthz-listen-address", opts.HealthzListenAddress, "listen address for healthz server")
	cmd.Flags().StringVar(&opts.MetricsListenAddress, "metrics-listen-address", opts.MetricsListenAddress, "listen address for metrics server")

	for _, flag := range []string{
		"config",
		"subscription-id",
		"resource-group",
		"storage-account-name",
		"kube-namespace",
	} {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			return err
		}
	}

	return nil
}

type validatedControllerOptions struct {
	*RawControllerOptions
	config             *config.Config
	cloudConfiguration cloud.Configuration
}

type ValidatedControllerOptions struct {
	*validatedControllerOptions
}

func (o *RawControllerOptions) Validate(ctx context.Context) (*ValidatedControllerOptions, error) {
	if len(o.ConfigPath) == 0 {
		return nil, fmt.Errorf("--config is required")
	}
	if len(o.SubscriptionID) == 0 {
		return nil, fmt.Errorf("--subscription-id is required")
	}
	if len(o.ResourceGroup) == 0 {
		return nil, fmt.Errorf("--resource-group is required")
	}
	if len(o.StorageAccountName) == 0 {
		return nil, fmt.Errorf("--storage-account-name is required")
	}
	if len(o.KubeNamespace) == 0 {
		return nil, fmt.Errorf("--kube-namespace is required")
	}
	if len(o.LeaderElectionID) == 0 {
		return nil, fmt.Errorf("--leader-election-id is required")
	}
	if o.InstallerCooldown <= 0 {
		return nil, fmt.Errorf("--installer-cooldown must be positive")
	}
	if o.MarketplaceCooldown <= 0 {
		return nil, fmt.Errorf("--marketplace-cooldown must be positive")
	}

	cfg, err := config.Load(o.ConfigPath)
	if err != nil {
		return nil, err
	}
	cloudConfig, err := azsdk.CloudConfigurationFromName(o.CloudEnvironment)
	if err != nil {
		return nil, fmt.Errorf("--cloud-environment: %w", err)
	}

	return &ValidatedControllerOptions{
		validatedControllerOptions: &validatedControllerOptions{
			RawControllerOptions: o,
			config:               cfg,
			cloudConfiguration:   cloudConfig,
		},
	}, nil
}

type controllerOptions struct {
	config          *config.Config
	configPath      string
	streamClient    *stream.Client
	versionLister   marketplace.VersionLister
	ingestionClient marketplace.IngestionClient
	storageClient   *storage.Client
	downloader      *download.Downloader
	statusClient    *status.Client

	leaderElectionLock resourcelock.Interface

	installerCooldown   time.Duration
	marketplaceCooldown time.Duration
	publishEnabled      bool

	healthzListenAddr string
	metricsListenAddr string
}

type ControllerOptions struct {
	*controllerOptions
}

func (o *ValidatedControllerOptions) Complete(ctx context.Context) (*ControllerOptions, error) {
	clientOpts := azsdk.NewClientOptions(azsdk.ComponentRHCOSPublisher)
	clientOpts.Cloud = o.cloudConfiguration

	credential, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
		ClientOptions:                clientOpts,
		RequireAzureTokenCredentials: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	versionLister, err := marketplace.NewARMVersionLister(o.SubscriptionID, o.MarketplaceLocation, credential, clientOpts)
	if err != nil {
		return nil, err
	}

	storageClient, err := storage.New(o.SubscriptionID, o.ResourceGroup, o.StorageAccountName, credential, clientOpts)
	if err != nil {
		return nil, err
	}

	var ingestionClient marketplace.IngestionClient
	if o.MarketplacePublishEnabled {
		ingestionClient = marketplace.NewHTTPIngestionClient(credential, "")
	}

	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster kubeconfig: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	leaderElectionLock, err := manager.NewLeaderElectionLock(hostname, kubeconfig, o.KubeNamespace, o.LeaderElectionID)
	if err != nil {
		return nil, err
	}

	return &ControllerOptions{
		controllerOptions: &controllerOptions{
			config:              o.config,
			configPath:          o.ConfigPath,
			streamClient:        stream.NewClient(""),
			versionLister:       versionLister,
			ingestionClient:     ingestionClient,
			storageClient:       storageClient,
			downloader:          download.New(o.WorkDir),
			statusClient:        status.NewClient(dynamicClient, o.KubeNamespace),
			leaderElectionLock:  leaderElectionLock,
			installerCooldown:   o.InstallerCooldown,
			marketplaceCooldown: o.MarketplaceCooldown,
			publishEnabled:      o.MarketplacePublishEnabled,
			healthzListenAddr:   o.HealthzListenAddress,
			metricsListenAddr:   o.MetricsListenAddress,
		},
	}, nil
}

func (o *ControllerOptions) Run(ctx context.Context) error {
	mgr := &manager.Manager{
		Config:              o.config,
		ConfigPath:          o.configPath,
		StreamClient:        o.streamClient,
		VersionLister:       o.versionLister,
		IngestionClient:     o.ingestionClient,
		StorageClient:       o.storageClient,
		Downloader:          o.downloader,
		StatusClient:        o.statusClient,
		LeaderElectionLock:  o.leaderElectionLock,
		InstallerCooldown:   o.installerCooldown,
		MarketplaceCooldown: o.marketplaceCooldown,
		PublishEnabled:      o.publishEnabled,
		HealthzListenAddr:   o.healthzListenAddr,
		MetricsListenAddr:   o.metricsListenAddr,
	}
	return mgr.Run(ctx)
}
