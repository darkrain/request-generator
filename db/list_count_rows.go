package db

// WithListCountRows opts into counting a filtered relation with its page.
// The producer must guarantee a unique, non-null primary key in the base
// relation. Joins, zero-size reads and unsupported executors retain separate
// COUNT behavior. This changes no permission or response contract.
func WithListCountRows(executor DBExecutor) DBExecutor {
	if scoped, ok := executor.(interface{ WithListCountRows() DBExecutor }); ok {
		return scoped.WithListCountRows()
	}
	return executor
}

func (db *DB) WithListCountRows() DBExecutor {
	scoped := *db
	scoped.listCountWithRows = true
	return &scoped
}
