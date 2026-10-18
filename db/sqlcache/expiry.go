package sqlcache

import (
	"container/heap"
	"time"
)

// Exactly one item per retained result, including during refresh. Removing a
// replaced item prevents an unbounded queue of obsolete refresh deadlines.
type expiryItem struct {
	key      [32]byte
	deadline time.Time
	index    int
}

type expiryQueue []*expiryItem

func (q expiryQueue) Len() int           { return len(q) }
func (q expiryQueue) Less(i, j int) bool { return q[i].deadline.Before(q[j].deadline) }
func (q expiryQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index, q[j].index = i, j
}
func (q *expiryQueue) Push(x any) {
	e := x.(*expiryItem)
	e.index = len(*q)
	*q = append(*q, e)
}
func (q *expiryQueue) Pop() any {
	old := *q
	e := old[len(old)-1]
	old[len(old)-1] = nil
	e.index = -1
	*q = old[:len(old)-1]
	return e
}

func minExpiry(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Caller holds c.mu. TTL is still checked independently on every lookup.
func (c *Cache) reclaimExpired(now time.Time) {
	for len(c.expirations) > 0 && !now.Before(c.expirations[0].deadline) {
		item := heap.Pop(&c.expirations).(*expiryItem)
		e := c.entries[item.key]
		delete(c.entries, item.key)
		c.bytes -= e.data.bytes
	}
}
