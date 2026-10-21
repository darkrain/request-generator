package db

// WithShortFirstPageCount returns an executor copy that derives the total
// from a non-full first List page without a second SELECT. The producer must
// guarantee one result per base row (no set-returning projections) and stable
// read predicates. Counts describe the same SQL result snapshot as the list,
// including its existing result-cache TTL, rather than a later COUNT snapshot.
// Joins and all other pages still execute COUNT. The shared executor and its
// caches are unchanged; no rows or authorization decisions are cached here.
func (db *DB) WithShortFirstPageCount() *DB {
	scoped := *db
	scoped.shortFirstPageCount = true
	return &scoped
}
