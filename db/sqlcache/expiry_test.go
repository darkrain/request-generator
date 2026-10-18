package sqlcache

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"
)

func TestExpiryQueueReclaimsBytesAndKeepsLiveResults(t *testing.T) {
	c := testCache()
	for i := 0; i < 16; i++ {
		rows, err := c.query(context.Background(), fmt.Sprintf("SELECT %d", i), nil, true, func() (driver.Rows, error) { return &testRows{n: i + 1}, nil })
		if err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	// Increasing the setting must preserve the original expiration deadlines.
	c.SetTTL(time.Hour)
	var first, last time.Time
	for _, e := range c.entries {
		if first.IsZero() || e.expires.Before(first) {
			first = e.expires
		}
		if last.Before(e.expires) {
			last = e.expires
		}
	}
	c.reclaimExpired(first)
	if len(c.entries) != 15 || len(c.expirations) != 15 {
		t.Fatal("wrong expiry boundary")
	}
	var retained int64
	for _, e := range c.entries {
		retained += e.data.bytes
	}
	if c.bytes != retained {
		t.Fatal("incorrect byte accounting")
	}
	c.reclaimExpired(last)
	if len(c.entries) != 0 || len(c.expirations) != 0 || c.bytes != 0 {
		t.Fatal("expired results retained")
	}
}

func TestExpiryQueueRefreshMetadataStaysBounded(t *testing.T) {
	c := testCache()
	c.options.MaxEntries = 1
	for i := 0; i < 1000; i++ {
		primeRefresh(t, c)
		if len(c.expirations) != 1 || len(c.entries) != 1 {
			t.Fatal("refresh metadata grew")
		}
		for _, e := range c.entries {
			if e.expiry.index != 0 || c.expirations[0] != e.expiry {
				t.Fatal("wrong replacement index")
			}
		}
	}
}

func TestExpiryQueueTTLChangeWhileReadInFlight(t *testing.T) {
	c := testCache()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		rows, err := c.query(context.Background(), "SELECT value", nil, true, func() (driver.Rows, error) {
			close(started)
			<-release
			return &testRows{n: 1}, nil
		})
		if rows != nil {
			rows.Close()
		}
		done <- err
	}()
	<-started
	c.SetTTL(time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for _, e := range c.entries {
		if !e.expiry.deadline.Equal(e.data.completed.Add(time.Millisecond)) {
			t.Fatal("in-flight result used old cleanup TTL")
		}
	}
	c.SetTTL(time.Nanosecond)
	c.reclaimExpired(time.Now())
	if len(c.entries) != 0 || c.bytes != 0 {
		t.Fatal("TTL reduction did not reclaim result")
	}
}

func TestExpiryQueueReplacesNonRootEntries(t *testing.T) {
	c := testCache()
	c.options.MaxEntries = 8
	c.SetTTL(time.Hour)
	for i := 0; i < 1000; i++ {
		query := fmt.Sprintf("SELECT %d", (i*5)%8)
		k, _ := key(query, nil)
		if e, ok := c.entries[k]; ok {
			e.refreshAt = time.Now().Add(-time.Second)
			c.entries[k] = e
		}
		rows, err := c.query(context.Background(), query, nil, true, func() (driver.Rows, error) { return &testRows{n: 1}, nil })
		if err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	if len(c.entries) != 8 || len(c.expirations) != 8 {
		t.Fatal("replacement grew or lost entries")
	}
	var last time.Time
	for _, e := range c.entries {
		if last.Before(e.expires) {
			last = e.expires
		}
	}
	c.reclaimExpired(last)
	if len(c.entries) != 0 || c.bytes != 0 {
		t.Fatal("replacement broke expiry ordering")
	}
}
