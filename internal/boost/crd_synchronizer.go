// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package boost

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	toolscache "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

// CRDSynchronizer synchronizes StartupCPUBoost CRDs into boost.Manager on non leader replicas
// where the reconcilers are not started. This enables mutating webhook operation on every
// replica.
type CRDSynchronizer interface {
	ctrlmgr.Runnable
	ctrlmgr.LeaderElectionRunnable
	HasSynced() bool
}

type crdSynchronizerImpl struct {
	mu               sync.RWMutex
	cache            ctrlcache.Cache
	client           client.Client
	mgr              Manager
	legacyRevertMode bool
	elected          <-chan struct{}
	informer         ctrlcache.Informer
	log              logr.Logger
}

// CRDSynchronizerConfig holds dependencies and configuration for CRDSynchronizer.
type CRDSynchronizerConfig struct {
	Client           client.Client
	Cache            ctrlcache.Cache
	Manager          Manager
	LegacyRevertMode bool
	Elected          <-chan struct{}
}

// NewCRDSynchronizer constructs a new CRDSynchronizer.
func NewCRDSynchronizer(cfg CRDSynchronizerConfig) CRDSynchronizer {
	return &crdSynchronizerImpl{
		client:           cfg.Client,
		cache:            cfg.Cache,
		mgr:              cfg.Manager,
		legacyRevertMode: cfg.LegacyRevertMode,
		elected:          cfg.Elected,
		log:              ctrl.Log.WithName("crd-synchronizer"),
	}
}

func (c *crdSynchronizerImpl) NeedLeaderElection() bool {
	return false
}

func (c *crdSynchronizerImpl) HasSynced() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *crdSynchronizerImpl) Start(ctx context.Context) error {
	c.log.Info("starting CRD synchronizer")
	informer, err := c.cache.GetInformer(ctx, &autoscaling.StartupCPUBoost{})
	if err != nil {
		return fmt.Errorf("failed to get StartupCPUBoost informer: %w", err)
	}
	c.mu.Lock()
	c.informer = informer
	c.mu.Unlock()

	registration, err := informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			c.onAdd(obj)
		},
		UpdateFunc: func(oldObj, newObj any) {
			c.onUpdate(newObj)
		},
		DeleteFunc: func(obj any) {
			c.onDelete(obj)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}
	defer func() {
		if err := informer.RemoveEventHandler(registration); err != nil {
			c.log.Error(err, "failed to remove event handler from shared informer")
		}
	}()
	select {
	case <-ctx.Done():
		c.log.Info("stopping CRD synchronizer on shutdown")
	case <-c.elected:
		c.log.Info("stopping CRD synchronizer as replica was elected leader")
	}
	return nil
}

func (c *crdSynchronizerImpl) onAdd(obj any) {
	boostObj, ok := obj.(*autoscaling.StartupCPUBoost)
	if !ok {
		return
	}
	log := c.log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost add from informer")
	boostCfg := &StartupCPUBoostConfig{
		Client:           c.client,
		LegacyRevertMode: c.legacyRevertMode,
	}
	boost, err := NewStartupCPUBoost(boostObj, boostCfg)
	if err != nil {
		log.Error(err, "boost creation error")
		return
	}
	if err := c.mgr.AddRegularCPUBoost(context.Background(), boost); err != nil {
		if errors.Is(err, ErrStartupCPUBoostAlreadyExists) {
			log.V(5).Info("boost already registered, updating")
			_ = c.mgr.UpdateRegularCPUBoost(context.Background(), boostObj)
		} else {
			log.Error(err, "boost registration error")
		}
	}
}

func (c *crdSynchronizerImpl) onUpdate(newObj any) {
	boostObj, ok := newObj.(*autoscaling.StartupCPUBoost)
	if !ok {
		return
	}
	log := c.log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost update from informer")
	if err := c.mgr.UpdateRegularCPUBoost(context.Background(), boostObj); err != nil {
		log.Error(err, "boost update error")
	}
}

func (c *crdSynchronizerImpl) onDelete(obj any) {
	boostObj, ok := obj.(*autoscaling.StartupCPUBoost)
	if !ok {
		tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		boostObj, ok = tombstone.Obj.(*autoscaling.StartupCPUBoost)
		if !ok {
			return
		}
	}
	log := c.log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost delete from informer")
	c.mgr.DeleteRegularCPUBoost(context.Background(), boostObj.Namespace, boostObj.Name)
}
