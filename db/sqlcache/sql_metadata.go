package sqlcache

import (
	"strings"
	"sync"
)

// Only pure functions of exact SQL bytes are retained here. Parameters,
// transaction/session state, bypass flags and result TTL are evaluated for
// every call as before. This is shared by prepared and direct reads.
type sqlMetadata struct {
	digest   [32]byte
	readOnly bool
}

const maxSQLMetadataEntries = 256
const maxSQLMetadataBytes = 4 << 20

type sqlMetadataCache struct {
	mu      sync.RWMutex
	entries map[string]sqlMetadata
	bytes   int
}

func (c *sqlMetadataCache) lookup(query string) sqlMetadata {
	c.mu.RLock()
	value, ok := c.entries[query]
	c.mu.RUnlock()
	if ok {
		return value
	}
	value = sqlMetadata{digest: hashSQL(query), readOnly: readOnly(query)}
	if len(query) > maxSQLMetadataBytes {
		return value
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if previous, ok := c.entries[query]; ok {
		return previous
	}
	// Hits require only a read lock. Resetting a full bounded metadata map is
	// safe: it changes computation cost, never the result cache or its expiry.
	if c.entries == nil || len(c.entries) >= maxSQLMetadataEntries || c.bytes+len(query) > maxSQLMetadataBytes {
		c.entries = make(map[string]sqlMetadata)
		c.bytes = 0
	}
	c.entries[strings.Clone(query)] = value
	c.bytes += len(query)
	return value
}
