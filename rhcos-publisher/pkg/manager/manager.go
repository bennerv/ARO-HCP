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

// Package manager wires the rhcos-publisher controllers together: it serves
// health and metrics endpoints and runs the controllers under a
// leader-election lease. A change of the mounted configuration file triggers
// a graceful shutdown so the Deployment restarts the process with the new
// configuration.
package manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	_ "k8s.io/component-base/metrics/prometheus/clientgo"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/component-base/metrics/legacyregistry"

	sharedleaderelection "github.com/Azure/ARO-HCP/internal/leaderelection"
	"github.com/Azure/ARO-HCP/internal/utils"
	"github.com/Azure/ARO-HCP/internal/version"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/config"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/base"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/installer"
	marketplacelister "github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/publisher"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/controllers/reconciler"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/download"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/marketplace"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/status"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/storage"
	"github.com/Azure/ARO-HCP/rhcos-publisher/pkg/stream"
)

const (
	name = "ARO HCP rhcos-publisher controller"

	healthzAdaptorTimeout   = 20 * time.Second
	httpServerShutdownTime  = 31 * time.Second
	configWatchInterval     = time.Minute
	orphanedBranchLogPrefix = "rhcos/"
)

// Manager runs the rhcos-publisher controllers.
type Manager struct {
	Config          *config.Config
	ConfigPath      string
	StreamClient    *stream.Client
	VersionLister   marketplace.VersionLister
	IngestionClient marketplace.IngestionClient // nil when marketplace publishing is disabled
	StorageClient   *storage.Client
	Downloader      *download.Downloader
	StatusClient    *status.Client

	LeaderElectionLock resourcelock.Interface

	InstallerCooldown   time.Duration
	MarketplaceCooldown time.Duration
	PublishEnabled      bool

	HealthzListenAddr string
	MetricsListenAddr string
}

// Run starts the manager. It serves /healthz and /metrics, then runs the
// controllers under a leader-election lease until ctx is cancelled or the
// configuration file changes.
func (m *Manager) Run(ctx context.Context) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Info("starting", "component", name, "commit", version.CommitSHA)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(fmt.Errorf("Run returned"))

	electionChecker := leaderelection.NewLeaderHealthzAdaptor(healthzAdaptorTimeout)

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	if len(m.HealthzListenAddr) > 0 {
		healthGauge := promauto.With(legacyregistry.Registerer()).NewGauge(prometheus.GaugeOpts{
			Name: "rhcos_publisher_health", Help: "rhcos_publisher_health is 1 when healthy",
		})
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if err := electionChecker.Check(r); err != nil {
				logger.V(1).Info("readiness probe failed", "error", err)
				http.Error(w, "lease not renewed", http.StatusServiceUnavailable)
				healthGauge.Set(0)
				return
			}
			w.WriteHeader(http.StatusOK)
			healthGauge.Set(1)
		})
		server := &http.Server{Addr: m.HealthzListenAddr, Handler: mux}
		wg.Add(1)
		go func() {
			defer utilruntime.HandleCrash()
			defer cancel(fmt.Errorf("healthz server exited"))
			defer wg.Done()
			if err := runHTTPServer(ctx, server, "healthz server"); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	if len(m.MetricsListenAddr) > 0 {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
			legacyregistry.Registerer(),
			promhttp.HandlerFor(prometheus.Gatherers{legacyregistry.DefaultGatherer}, promhttp.HandlerOpts{}),
		))
		server := &http.Server{Addr: m.MetricsListenAddr, Handler: mux}
		wg.Add(1)
		go func() {
			defer utilruntime.HandleCrash()
			defer cancel(fmt.Errorf("metrics server exited"))
			defer wg.Done()
			if err := runHTTPServer(ctx, server, "metrics server"); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer utilruntime.HandleCrash()
		defer cancel(fmt.Errorf("leader election exited"))
		defer wg.Done()
		if err := m.runControllersUnderLeaderElection(ctx, cancel, electionChecker); err != nil {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
	}()

	wg.Wait()
	logger.Info("stopped", "component", name, "commit", version.CommitSHA)
	return errors.Join(errs...)
}

func (m *Manager) runControllersUnderLeaderElection(
	ctx context.Context, shutdown context.CancelCauseFunc, electionChecker *leaderelection.HealthzAdaptor,
) error {
	logger := utils.LoggerFromContext(ctx)

	// The listers enqueue into the reconciler's queue and the reconciler into
	// the publisher's queue. The listers reference the reconciler controller
	// through a closure since it is created after them.
	var reconcilerController *base.Controller

	installerSyncer := installer.NewSyncer(m.StreamClient, m.StatusClient, m.Config, func(ctx context.Context, key string) {
		reconcilerController.Enqueue(ctx, key)
	})
	marketplaceSyncer := marketplacelister.NewSyncer(m.VersionLister, m.StatusClient, m.Config, func(ctx context.Context, key string) {
		reconcilerController.Enqueue(ctx, key)
	})

	var publisherController *base.Controller
	enqueuePublish := func(context.Context, string) {}
	if m.PublishEnabled {
		publisherSyncer := publisher.NewSyncer(m.IngestionClient, installerSyncer, m.StorageClient, m.StatusClient, m.Config)
		publisherController = base.NewController(publisher.ControllerName, publisherSyncer, 0)
		enqueuePublish = publisherController.Enqueue
	}

	reconcilerSyncer := reconciler.NewSyncer(
		installerSyncer,
		marketplaceSyncer,
		m.StorageClient,
		m.Downloader,
		m.StatusClient,
		m.Config,
		m.PublishEnabled,
		enqueuePublish,
	)
	reconcilerController = base.NewController(reconciler.ControllerName, reconcilerSyncer, 0)

	installerController := base.NewController(installer.ControllerName, installerSyncer, m.InstallerCooldown)
	marketplaceController := base.NewController(marketplacelister.ControllerName, marketplaceSyncer, m.MarketplaceCooldown)

	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock:          m.LeaderElectionLock,
		LeaseDuration: sharedleaderelection.RecommendedLeaseDuration,
		RenewDeadline: sharedleaderelection.RecommendedRenewDeadline,
		RetryPeriod:   sharedleaderelection.RecommendedRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Info("acquired leader election lease; starting controllers")

				// A config change requires re-validating and re-wiring
				// everything; restart the process gracefully.
				configWatcher, err := utils.NewFSWatcher(m.ConfigPath, configWatchInterval, func(ctx context.Context) error {
					shutdown(fmt.Errorf("configuration file %s changed", m.ConfigPath))
					return nil
				})
				if err != nil {
					shutdown(fmt.Errorf("failed to create config watcher: %w", err))
					return
				}
				if err := configWatcher.Start(ctx); err != nil {
					shutdown(fmt.Errorf("failed to start config watcher: %w", err))
					return
				}

				if err := m.cleanupOrphanedBranches(ctx); err != nil {
					// Best effort: orphans do not block publishing new images.
					logger.Error(err, "failed to clean up orphaned staged VHDs")
				}

				go installerController.Run(ctx, 1)
				go marketplaceController.Run(ctx, 1)
				go reconcilerController.Run(ctx, 1)
				if publisherController != nil {
					go publisherController.Run(ctx, 1)
				}

				go runTicker(ctx, m.InstallerCooldown, func() { installerController.Enqueue(ctx, installer.SyncKey) })
				go runTicker(ctx, m.MarketplaceCooldown, func() { marketplaceController.Enqueue(ctx, marketplacelister.SyncKey) })
			},
			OnStoppedLeading: func() {
				logger.Info("lost leader election lease")
			},
		},
		ReleaseOnCancel: true,
		WatchDog:        electionChecker,
		Name:            "rhcos-publisher-controller",
	}

	sharedleaderelection.LogLeaseProperties(logger, leaderElectionConfig)

	leaderElector, err := leaderelection.NewLeaderElector(leaderElectionConfig)
	if err != nil {
		return err
	}
	leaderElector.Run(ctx)
	return nil
}

// cleanupOrphanedBranches purges staged VHDs of branches that are no longer
// configured (branch removal takes effect via config change + restart).
func (m *Manager) cleanupOrphanedBranches(ctx context.Context) error {
	logger := utils.LoggerFromContext(ctx)

	blobPaths, err := m.StorageClient.ListBlobs(ctx, orphanedBranchLogPrefix)
	if err != nil {
		return err
	}
	var errs []error
	for _, blobPath := range blobPaths {
		branch, ok := storage.BranchOfBlobPath(blobPath)
		if !ok {
			continue
		}
		if _, configured := m.Config.Branch(branch); configured {
			continue
		}
		if err := m.StorageClient.DeleteBlob(ctx, blobPath); err != nil {
			errs = append(errs, err)
			continue
		}
		logger.Info("purged staged VHD of removed branch", "blob", blobPath)
	}
	return errors.Join(errs...)
}

// runTicker enqueues via enqueue immediately and then on every tick until ctx
// is cancelled.
func runTicker(ctx context.Context, interval time.Duration, enqueue func()) {
	defer utilruntime.HandleCrash()

	enqueue()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			enqueue()
		case <-ctx.Done():
			return
		}
	}
}

// runHTTPServer runs the server and shuts it down when ctx is cancelled.
// It returns nil if the server was shut down cleanly (http.ErrServerClosed),
// or the underlying error if ListenAndServe failed for another reason.
func runHTTPServer(ctx context.Context, server *http.Server, name string) error {
	logger := utils.LoggerFromContext(ctx)

	done := make(chan struct{})
	defer close(done)
	go func() {
		defer utilruntime.HandleCrash()
		select {
		case <-ctx.Done():
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), httpServerShutdownTime)
			defer shutdownCancel()
			logger.Info("shutting down server", "server", name)
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error(err, "failed to shut down server", "server", name)
			} else {
				logger.Info("server shut down completed", "server", name)
			}
		case <-done:
		}
	}()

	logger.Info("server listening", "server", name, "address", server.Addr)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
