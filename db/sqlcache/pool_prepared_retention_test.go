package sqlcache

import (
	"context"
	"fmt"
	"testing"
)

// Logical connections are cheap cache lookups; their turnover must not throw
// away the server plans owned by the long-lived physical pool.
func TestPoolPreparedPlanSurvivesLogicalConnectionTurnover(t *testing.T) {
	_, front, backend := postgresPool(t, 1)
	front.SetMaxIdleConns(0)
	if _, err := backend.Exec("SET plan_cache_mode=force_generic_plan"); err != nil {
		t.Fatal(err)
	}
	const query = "SELECT $1::bigint AS retained_plan_probe"
	stmt, err := front.Prepare(query)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	for i := int64(0); i < 8; i++ {
		var got int64
		if err := stmt.QueryRowContext(WithoutCache(context.Background()), i).Scan(&got); err != nil || got != i {
			t.Fatalf("value=%d err=%v", got, err)
		}
	}
	var plans int
	if err := backend.QueryRow("SELECT COALESCE(SUM(generic_plans),0) FROM pg_prepared_statements WHERE statement=$1", query).Scan(&plans); err != nil {
		t.Fatal(err)
	}
	if plans != 8 {
		t.Fatalf("server plan was discarded during logical turnover: executions=%d, want 8", plans)
	}
	if err := front.Close(); err != nil {
		t.Fatal(err)
	}
	if err := backend.QueryRow("SELECT COUNT(*) FROM pg_prepared_statements WHERE statement=$1", query).Scan(&plans); err != nil || plans != 0 {
		t.Fatalf("closing logical pool did not release retained plans: count=%d err=%v", plans, err)
	}
}

func TestPoolPreparedRetentionBoundsIdlePlansAndDrainsActiveReferences(t *testing.T) {
	_, _, backend := postgresPool(t, 1)
	p := &poolConnector{backend: backend, statements: make(map[string]*poolStatement)}
	t.Cleanup(func() { p.Close() })
	ctx := context.Background()
	const activeQuery = "SELECT $1::bigint AS retention_active"
	active, err := p.prepare(ctx, activeQuery)
	if err != nil {
		t.Fatal(err)
	}
	query := func(i int) string { return fmt.Sprintf("SELECT $1::bigint AS retention_bound_%d", i) }
	prepareAndRelease := func(i int) {
		t.Helper()
		s, err := p.prepare(ctx, query(i))
		if err != nil {
			t.Fatal(err)
		}
		p.release(query(i), s)
	}
	for i := 0; i <= maxIdlePoolStatements; i++ {
		prepareAndRelease(i)
	}
	prepareAndRelease(1) // Touch an old entry, then force another eviction.
	prepareAndRelease(maxIdlePoolStatements + 1)
	var count int
	if err := backend.QueryRow("SELECT COUNT(*) FROM pg_prepared_statements").Scan(&count); err != nil || count != maxIdlePoolStatements+1 {
		t.Fatalf("retained plan count=%d err=%v", count, err)
	}
	for i, want := range map[int]int{0: 0, 1: 1, 2: 0} {
		if err := backend.QueryRow("SELECT COUNT(*) FROM pg_prepared_statements WHERE statement=$1", query(i)).Scan(&count); err != nil || count != want {
			t.Fatalf("retained query %d: count=%d want=%d err=%v", i, count, want, err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	// Closing the frontend must not abort an already checked-out statement.
	if err := active.stmt.QueryRow(int64(17)).Scan(&count); err != nil || count != 17 {
		t.Fatalf("active reference interrupted: value=%d err=%v", count, err)
	}
	p.release(activeQuery, active)
	if err := backend.QueryRow("SELECT COUNT(*) FROM pg_prepared_statements").Scan(&count); err != nil || count != 0 {
		t.Fatalf("plans leaked after drain: count=%d err=%v", count, err)
	}
}
