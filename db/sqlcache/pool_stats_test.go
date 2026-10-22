package sqlcache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPoolPlanStatsSeparateReuseEvictionAndPrepareErrors(t *testing.T) {
	backend, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	cache := New(Options{})
	p := &poolConnector{backend: backend, cache: cache, statements: make(map[string]*poolStatement)}
	ctx := context.Background()
	for i := 0; i <= maxIdlePoolStatements; i++ {
		query := fmt.Sprintf("SELECT %d AS probe", i)
		mock.ExpectPrepare(query)
		s, err := p.prepare(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		p.release(query, s)
	}
	query := fmt.Sprintf("SELECT %d AS probe", maxIdlePoolStatements)
	s, err := p.prepare(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	p.release(query, s)
	mock.ExpectPrepare("SELECT broken").WillReturnError(errors.New("prepare failed"))
	if _, err := p.prepare(ctx, "SELECT broken"); err == nil {
		t.Fatal("expected prepare error")
	}
	stats := cache.Stats()
	if stats.PlanPrepares != maxIdlePoolStatements+2 || stats.PlanReuses != 1 ||
		stats.PlanEvictions != 1 || stats.PlanErrors != 1 {
		t.Fatalf("unexpected plan stats: %+v", stats)
	}
	if stats.Misses != 0 || stats.Hits != 0 {
		t.Fatal("plan preparation must not count as result lookup")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if cache.Stats().PlanEvictions != 1 {
		t.Fatal("pool shutdown is not capacity eviction")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
