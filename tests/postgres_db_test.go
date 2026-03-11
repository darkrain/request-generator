package tests

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/go-jet/jet/v2/postgres"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func execSQLFile(db *sql.DB, filename string) error {
	path := filepath.Join(testDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(data))
	return err
}

type testTable struct {
	postgres.Table

	ID    postgres.ColumnInteger
	Name  postgres.ColumnString
	Email postgres.ColumnString
	Age   postgres.ColumnInteger
	Role  postgres.ColumnString
}

func newTestTable() testTable {
	id := postgres.IntegerColumn("id")
	name := postgres.StringColumn("name")
	email := postgres.StringColumn("email")
	age := postgres.IntegerColumn("age")
	role := postgres.StringColumn("role")
	all := postgres.ColumnList{id, name, email, age, role}

	return testTable{
		Table: postgres.NewTable("public", "test_items", "", all...),
		ID:    id,
		Name:  name,
		Email: email,
		Age:   age,
		Role:  role,
	}
}

var (
	sqlDB   *sql.DB
	testDB  *db.DB
	tbl     testTable
	testLog *log.Entry
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost port=5432 user=postgres password=postgres dbname=iamfree sslmode=disable"
	}

	var err error
	sqlDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Warn("Skipping DB tests: ", err)
		os.Exit(0)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Warn("Skipping DB tests: cannot ping: ", err)
		os.Exit(0)
	}

	testDB = db.NewDB(sqlDB)
	testDB.Debug = true
	tbl = newTestTable()
	testLog = log.WithFields(log.Fields{"test": true})

	if err = execSQLFile(sqlDB, "setup.sql"); err != nil {
		log.Fatal("Cannot execute setup.sql: ", err)
	}

	code := m.Run()

	execSQLFile(sqlDB, "teardown.sql")
	sqlDB.Close()

	os.Exit(code)
}

func cleanTable(t *testing.T) {
	_, err := sqlDB.Exec("DELETE FROM test_tags")
	require.NoError(t, err)
	_, err = sqlDB.Exec("DELETE FROM test_items")
	require.NoError(t, err)
	_, err = sqlDB.Exec("ALTER SEQUENCE test_items_id_seq RESTART WITH 1")
	require.NoError(t, err)
}

func testModuleFields() []fields.ModuleField {
	t := newTestTable()
	return []fields.ModuleField{
		{Column: t.Name, Title: "Name", Type: fields.ModuleFieldTypeString},
		{Column: t.Email, Title: "Email", Type: fields.ModuleFieldTypeString},
		{Column: t.Age, Title: "Age", Type: fields.ModuleFieldTypeInt},
		{Column: t.Role, Title: "Role", Type: fields.ModuleFieldTypeString},
	}
}

func seedItem(t *testing.T, name, email string, age int, role string) int64 {
	mf := testModuleFields()
	input := map[string]interface{}{
		"name":  name,
		"email": email,
		"age":   age,
		"role":  role,
	}

	result, err := testDB.Add(testLog, tbl, tbl.ID, mf, input)
	require.NoError(t, err)

	output := result.(struct {
		Value      int64  `json:"value"`
		PrimaryKey string `json:"primary_key"`
	})
	return output.Value
}

// --- Add ---

func TestAdd(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@test.com",
		"age":   25,
		"role":  "admin",
	}

	result, err := testDB.Add(testLog, tbl, tbl.ID, mf, input)
	require.NoError(t, err)
	assert.NotNil(t, result)

	output := result.(struct {
		Value      int64  `json:"value"`
		PrimaryKey string `json:"primary_key"`
	})
	assert.Equal(t, int64(1), output.Value)
	assert.Equal(t, "id", output.PrimaryKey)
}

func TestAddDuplicateEmail(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input := map[string]interface{}{
		"name":  "Alice",
		"email": "dup@test.com",
		"age":   25,
		"role":  "user",
	}

	_, err := testDB.Add(testLog, tbl, tbl.ID, mf, input)
	require.NoError(t, err)

	_, err = testDB.Add(testLog, tbl, tbl.ID, mf, input)
	assert.Error(t, err)
}

func TestAddMissingRequiredField(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input := map[string]interface{}{
		"email": "noname@test.com",
		"age":   30,
	}

	_, err := testDB.Add(testLog, tbl, tbl.ID, mf, input)
	assert.Error(t, err)
}

// --- View ---

func TestView(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Bob", "bob@test.com", 30, "admin")

	mf := testModuleFields()
	where := postgres.RawBool(`test_items."id" = #id`, postgres.RawArgs{"#id": 1})

	result, err := testDB.View(testLog, tbl, tbl.ID, mf, where, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)

	row := result.(map[string]interface{})
	assert.Equal(t, "Bob", row["name"])
	assert.Equal(t, "bob@test.com", row["email"])
	assert.Equal(t, int64(30), row["age"])
	assert.Equal(t, "admin", row["role"])
}

func TestViewNotFound(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	where := postgres.RawBool(`test_items."id" = #id`, postgres.RawArgs{"#id": 999})

	_, err := testDB.View(testLog, tbl, tbl.ID, mf, where, nil)
	assert.Error(t, err)
}

// --- List ---

func TestListEmpty(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, results)
}

func TestListMultipleItems(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "admin")
	seedItem(t, "Bob", "bob@test.com", 30, "user")
	seedItem(t, "Charlie", "charlie@test.com", 35, "user")

	mf := testModuleFields()
	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.Len(t, results, 3)
}

func TestListPagination(t *testing.T) {
	cleanTable(t)
	seedItem(t, "A", "a@test.com", 1, "user")
	seedItem(t, "B", "b@test.com", 2, "user")
	seedItem(t, "C", "c@test.com", 3, "user")

	mf := testModuleFields()

	results, _, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 2, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results, _, err = testDB.List(testLog, tbl, tbl.ID, mf, 1, 2, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestListSearch(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "admin")
	seedItem(t, "Bob", "bob@test.com", 30, "user")

	mf := testModuleFields()
	searchColumns := []postgres.Column{tbl.Email}

	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, searchColumns, "alice", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, results, 1)
}

func TestListFilter(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "admin")
	seedItem(t, "Bob", "bob@test.com", 30, "user")
	seedItem(t, "Charlie", "charlie@test.com", 35, "user")

	mf := testModuleFields()
	filter := map[string]string{"role": "user"}

	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", filter, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, results, 2)
}

func TestListWhere(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "admin")
	seedItem(t, "Bob", "bob@test.com", 30, "user")

	mf := testModuleFields()
	where := postgres.RawBool(`test_items."age" > #age`, postgres.RawArgs{"#age": 26})

	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, where, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, results, 1)
}

// --- Update ---

func TestUpdate(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "user")

	mf := testModuleFields()
	input := map[string]interface{}{
		"name": "Alice Updated",
		"age":  26,
	}
	where := postgres.RawBool(`"id" = #id`, postgres.RawArgs{"#id": 1})

	result, err := testDB.Update(testLog, tbl, tbl.ID, mf, input, where)
	require.NoError(t, err)
	assert.NotNil(t, result)

	row := result.(map[string]interface{})
	assert.Equal(t, "Alice Updated", row["name"])
	assert.Equal(t, int64(26), row["age"])
}

func TestUpdateNotFound(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input := map[string]interface{}{"name": "Ghost"}
	where := postgres.RawBool(`"id" = #id`, postgres.RawArgs{"#id": 999})

	_, err := testDB.Update(testLog, tbl, tbl.ID, mf, input, where)
	assert.Error(t, err)
}

func TestUpdateNilWhere(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input := map[string]interface{}{"name": "Bad"}

	_, err := testDB.Update(testLog, tbl, tbl.ID, mf, input, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WHERE condition is required")
}

// --- Delete ---

func TestDelete(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "user")

	where := postgres.RawBool(`"id" = #id`, postgres.RawArgs{"#id": 1})

	err := testDB.Delete(testLog, tbl, where)
	require.NoError(t, err)

	mf := testModuleFields()
	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Empty(t, results)
}

func TestDeleteNotFound(t *testing.T) {
	cleanTable(t)

	where := postgres.RawBool(`"id" = #id`, postgres.RawArgs{"#id": 999})

	err := testDB.Delete(testLog, tbl, where)
	assert.Error(t, err)
}

// --- Join ---

func TestListWithJoin(t *testing.T) {
	cleanTable(t)
	seedItem(t, "Alice", "alice@test.com", 25, "admin")

	_, err := sqlDB.Exec(`INSERT INTO test_tags (item_id, tag) VALUES (1, 'go'), (1, 'rust')`)
	require.NoError(t, err)

	tagID := postgres.IntegerColumn("id")
	tagItemID := postgres.IntegerColumn("item_id")
	tagCol := postgres.StringColumn("tag")
	tagCols := postgres.ColumnList{tagID, tagItemID, tagCol}
	tagsTable := postgres.NewTable("public", "test_tags", "tags", tagCols...)

	join := actions.ModuleActionJoin{
		Table:           tagsTable,
		Type:            actions.JoinTypeLeft,
		OnCondition:     postgres.RawBool(`test_items."id" = tags."item_id"`, nil),
		Columns:         []postgres.Column{tagCol},
		ResultArrayName: "tags",
	}

	mf := testModuleFields()
	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, nil, []actions.ModuleActionJoin{join})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, results, 1)

	row := results[0].(map[string]interface{})
	tags, ok := row["tags"]
	assert.True(t, ok)
	assert.NotNil(t, tags)
}

// --- Transaction rollback ---

func TestAddRollbackOnError(t *testing.T) {
	cleanTable(t)

	mf := testModuleFields()
	input1 := map[string]interface{}{"name": "First", "email": "first@test.com", "age": 20, "role": "user"}
	_, err := testDB.Add(testLog, tbl, tbl.ID, mf, input1)
	require.NoError(t, err)

	input2 := map[string]interface{}{"name": "Second", "email": "first@test.com", "age": 25, "role": "user"}
	_, err = testDB.Add(testLog, tbl, tbl.ID, mf, input2)
	assert.Error(t, err)

	results, count, err := testDB.List(testLog, tbl, tbl.ID, mf, 0, 100, nil, "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, results, 1)
}
