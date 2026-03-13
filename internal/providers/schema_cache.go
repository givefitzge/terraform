// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
)

// SchemaCache is a global cache of Schemas.
// This will be accessed by both core and the provider clients to ensure that
// large schemas are stored in a single location.
var SchemaCache = &schemaCache{
	m: make(map[addrs.Provider]map[versions.Version]ProviderSchema),
}

// Global cache for provider schemas
// Cache the entire response to ensure we capture any new fields, like
// ServerCapabilities. This also serves to capture errors so that multiple
// concurrent calls resulting in an error can be handled in the same manner.
type schemaCache struct {
	mu sync.Mutex
	m  map[addrs.Provider]map[versions.Version]ProviderSchema
}

func (c *schemaCache) Set(p addrs.Provider, v versions.Version, s ProviderSchema) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.m[p] == nil {
		c.m[p] = make(map[versions.Version]ProviderSchema)
	}
	c.m[p][v] = s
}

func (c *schemaCache) Get(p addrs.Provider, v versions.Version) (ProviderSchema, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for whether the provider exists in the cache at all
	vMap, ok := c.m[p]
	if !ok {
		return ProviderSchema{}, false
	}

	// Try to access the schema for the specific version
	s, ok := vMap[v]
	return s, ok
}
