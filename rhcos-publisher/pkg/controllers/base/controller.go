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

// Package base provides the workqueue controller shared by the rhcos-publisher
// controllers. It mirrors the fleet StampWatchingController, but the keys are
// plain strings and the queue is fed by tickers and by other controllers
// rather than by informers: the listers poll external systems (GitHub, the
// Azure Marketplace) and the reconciler and publisher are purely event-driven.
package base

import (
	"context"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/Azure/ARO-HCP/internal/controllerutils"
	"github.com/Azure/ARO-HCP/internal/utils"
)

// Syncer is the interface concrete controllers implement.
type Syncer interface {
	SyncOnce(ctx context.Context, key string) error
}

// Controller wraps a Syncer with a rate-limited workqueue and an optional
// cooldown gate.
type Controller struct {
	name     string
	syncer   Syncer
	queue    workqueue.TypedRateLimitingInterface[string]
	cooldown controllerutils.CooldownChecker
}

// NewController creates a controller. When cooldownPeriod is positive,
// Enqueue drops keys that were already allowed within the cooldown window;
// keys re-queued by the rate limiter after a sync error bypass the gate.
func NewController(name string, syncer Syncer, cooldownPeriod time.Duration) *Controller {
	var cooldown controllerutils.CooldownChecker
	if cooldownPeriod > 0 {
		cooldown = controllerutils.NewTimeBasedCooldownChecker(cooldownPeriod)
	}
	return &Controller{
		name:   name,
		syncer: syncer,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: name},
		),
		cooldown: cooldown,
	}
}

// Enqueue adds a key to the workqueue, subject to the cooldown gate. It is
// the entry point for tickers and for cross-controller signaling.
func (c *Controller) Enqueue(ctx context.Context, key string) {
	if c.cooldown != nil && !c.cooldown.CanSync(ctx, key) {
		return
	}
	c.queue.Add(key)
}

// Run processes the workqueue with the given number of workers until ctx is
// cancelled.
func (c *Controller) Run(ctx context.Context, threadiness int) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	ctx = utils.ContextWithControllerName(ctx, c.name)
	logger := utils.LoggerFromContext(ctx).WithValues(utils.LogValues{}.AddControllerName(c.name)...)
	ctx = utils.ContextWithLogger(ctx, logger)
	logger.Info("starting controller")
	defer logger.Info("stopped controller")

	for range threadiness {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	<-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNext(ctx) {
	}
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	logger := utils.LoggerFromContext(ctx).WithValues("key", key)
	ctx = utils.ContextWithLogger(ctx, logger)

	ReconcileTotal.WithLabelValues(c.name).Inc()
	if err := c.syncer.SyncOnce(ctx, key); err != nil {
		ReconcileErrorsTotal.WithLabelValues(c.name).Inc()
		utilruntime.HandleErrorWithContext(ctx, err, "sync error; requeuing", "key", key)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}
