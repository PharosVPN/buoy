// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

package awg

import (
	"context"
	"fmt"
	"sync"
)

// Registry owns the node's managed AmneziaWG interfaces. The client interface
// (Primary, awg0) is created at startup and serves end-user tunnels. Cascade
// inner links (DESIGN §3) are added on demand when the controller provisions a
// node→node edge and removed when it is torn down. It is safe for concurrent
// use; each Manager remains independently locked.
type Registry struct {
	primaryName string

	mu       sync.Mutex
	managers map[string]*Manager
}

// NewRegistry returns a Registry seeded with the primary client-interface
// Manager (whose Interface() name becomes the primary).
func NewRegistry(primary *Manager) *Registry {
	return &Registry{
		primaryName: primary.Interface(),
		managers:    map[string]*Manager{primary.Interface(): primary},
	}
}

// Primary returns the client-interface Manager (awg0).
func (r *Registry) Primary() *Manager {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.managers[r.primaryName]
}

// Get returns the Manager for iface, if registered.
func (r *Registry) Get(iface string) (*Manager, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.managers[iface]
	return m, ok
}

// Add registers a Manager for a new interface. It errors if one is already
// registered for that name — callers Get an existing one, or Remove then Add to
// replace it.
func (r *Registry) Add(m *Manager) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := m.Interface()
	if _, exists := r.managers[name]; exists {
		return fmt.Errorf("awg: interface %q already registered", name)
	}
	r.managers[name] = m
	return nil
}

// Remove tears the interface down and unregisters it. Removing the primary is
// rejected; removing an unknown interface is a no-op.
func (r *Registry) Remove(ctx context.Context, iface string) error {
	if iface == r.primaryName {
		return fmt.Errorf("awg: refusing to remove the primary interface %q", iface)
	}
	r.mu.Lock()
	m, ok := r.managers[iface]
	if ok {
		delete(r.managers, iface)
	}
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return m.Down(ctx)
}

// All returns every registered Manager. Order is unspecified.
func (r *Registry) All() []*Manager {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Manager, 0, len(r.managers))
	for _, m := range r.managers {
		out = append(out, m)
	}
	return out
}
