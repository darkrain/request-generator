package module

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// newDBErrorRouter is the router of darkrain/request-generator#169 and #167:
// a real gin engine with gin.Recovery and one module "items".
func newDBErrorRouter(t *testing.T, withLogger bool, configure func(*Generator, *BaseModule)) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pool, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	executor := db.NewDB(pool)

	id := pg.IntegerColumn("id")
	name := pg.StringColumn("name")
	ownerID := pg.IntegerColumn("owner_id")
	table := pg.NewTable("public", "items", "", id, name, ownerID)
	onlyOwner42 := func(*gin.Context) pg.BoolExpression { return ownerID.EQ(pg.Int(42)) }
	mod := &BaseModule{
		Name: "items", Path: "/api", Table: table, PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden},
			{Column: name, Title: "name", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		RoleWhere: []actions.RoleWhere{{Role: actions.RoleAll, Where: onlyOwner42}},
		Actions: []actions.ModuleAction{
			actions.AddModuleAction{Label: "add", Columns: []pg.Column{name}},
			actions.ViewModuleAction{Label: "view", Columns: []pg.Column{id, name}, By: []pg.Column{id}},
			actions.UpdateModuleAction{Label: "update", Columns: []pg.Column{name}, By: []pg.Column{id}, Where: onlyOwner42},
			actions.DeleteModuleAction{Label: "delete", By: []pg.Column{id}, Where: onlyOwner42},
		},
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	if withLogger {
		engine.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), icontext.LoggerContextKey, log.NewEntry(log.New())))
		})
	}
	gen := NewGenerator(func(*BaseModule) db.DBExecutor { return executor }, *engine.Group(""), []*BaseModule{mod}, nil, nil)
	if configure != nil {
		configure(gen, mod)
	}
	gen.Run()
	return engine, mock
}

func doDBErrorRequest(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "ru")
	engine.ServeHTTP(w, req)
	return w
}

type dbErrorBody struct {
	StatusCode int      `json:"statusCode"`
	Message    string   `json:"message"`
	Errors     []string `json:"errors"`
	Code       string   `json:"code"`
}

func decodeDBErrorBody(t *testing.T, w *httptest.ResponseRecorder) dbErrorBody {
	t.Helper()
	var body dbErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %q", w.Body.String())
	}
	return body
}

func assertNoDriverText(t *testing.T, w *httptest.ResponseRecorder, secrets ...string) {
	t.Helper()
	for _, secret := range append([]string{"pq:", "SQLSTATE", "relation", "violates"}, secrets...) {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("response exposes %q: %s", secret, w.Body.String())
		}
	}
}

var (
	checkViolation      = &pq.Error{Code: "23514", Message: `new row for relation "items" violates check constraint "items_name_check"`, Constraint: "items_name_check", Table: "items"}
	uniqueViolation     = &pq.Error{Code: "23505", Message: `duplicate key value violates unique constraint "items_name_key"`, Constraint: "items_name_key", Table: "items"}
	foreignKeyViolation = &pq.Error{Code: "23503", Message: `update or delete on table "items" violates foreign key constraint "orders_item_id_fkey" on table "orders"`, Constraint: "orders_item_id_fkey", Table: "items"}
	notNullViolation    = &pq.Error{Code: "23502", Message: `null value in column "name" of relation "items" violates not-null constraint`, Column: "name", Table: "items"}
	undefinedColumn     = &pq.Error{Code: "42703", Message: `column "secret_column" does not exist`}
)

// The reproduction of darkrain/request-generator#169: a check violation on
// add answered with the driver's text, relation and constraint names.
func TestAddDoesNotLeakDriverError(t *testing.T) {
	engine, mock := newDBErrorRouter(t, true, nil)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO public."items"`).WillReturnError(checkViolation)
	mock.ExpectRollback()
	w := doDBErrorRequest(engine, http.MethodPut, "/api/items", `{"name":"x"}`)
	assertNoDriverText(t, w, "items_name_check")
	body := decodeDBErrorBody(t, w)
	if w.Code != http.StatusBadRequest || body.Code != DBErrorCodeInvalidValue || body.Message != GeneratorErrorAdd {
		t.Fatalf("unexpected answer %d %+v", w.Code, body)
	}
	if len(body.Errors) != 1 || body.Errors[0] != dbErrorFallbacks[DBErrorKeyInvalidValue] {
		t.Fatalf("unexpected reason %+v", body.Errors)
	}
}

func TestDBErrorsMapToStableCodes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"unique", uniqueViolation, http.StatusConflict, DBErrorCodeConflict},
		{"foreign key", foreignKeyViolation, http.StatusBadRequest, DBErrorCodeReference},
		{"not null", notNullViolation, http.StatusBadRequest, DBErrorCodeRequired},
		{"check", checkViolation, http.StatusBadRequest, DBErrorCodeInvalidValue},
		{"invalid uuid", &pq.Error{Code: "22P02", Message: `invalid input syntax for type uuid: "x"`}, http.StatusBadRequest, DBErrorCodeInvalidValue},
		{"other", undefinedColumn, http.StatusInternalServerError, DBErrorCodeInternal},
		{"wrapped", fmt.Errorf("save item: %w", uniqueViolation), http.StatusConflict, DBErrorCodeConflict},
		{"pgx", &fakePgError{Code: "23505", ConstraintName: "items_name_key"}, http.StatusConflict, DBErrorCodeConflict},
		{"bad connection", fmt.Errorf("begin: %w", sql.ErrConnDone), http.StatusInternalServerError, DBErrorCodeInternal},
		{"driver text", errors.New("pq: could not complete operation in a failed transaction"), http.StatusInternalServerError, DBErrorCodeInternal},
		{"no rows", sql.ErrNoRows, http.StatusNotFound, DBErrorCodeNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			classified, ok := ClassifyDBError(tc.err)
			if !ok || classified.Status != tc.status || classified.Code != tc.code {
				t.Fatalf("got %+v, %v", classified, ok)
			}
		})
	}
	if _, ok := ClassifyDBError(actions.NewStatusError(http.StatusForbidden, "Нельзя")); ok {
		t.Fatal("a refusal of a hook is not a database error")
	}
	if _, ok := ClassifyDBError(errors.New("Это фото уже есть в галерее")); ok {
		t.Fatal("an application message is not a database error")
	}
	if got := constraintName(&fakePgError{ConstraintName: "x_key"}); got != "x_key" {
		t.Fatalf("pgx constraint = %q", got)
	}
}

// fakePgError has the shape of pgx's *pgconn.PgError.
type fakePgError struct {
	Code           string
	ConstraintName string
}

func (e *fakePgError) Error() string    { return "ERROR: duplicate key (SQLSTATE " + e.Code + ")" }
func (e *fakePgError) SQLState() string { return e.Code }

func TestUniqueViolationUsesModuleTranslation(t *testing.T) {
	engine, mock := newDBErrorRouter(t, true, func(gen *Generator, mod *BaseModule) {
		gen.Locales = []locale.Lang{locale.EN, locale.RU}
		gen.SetTranslationOverrides(locale.RU, map[string]string{
			"items.errors.name_taken": "Такое имя уже есть",
			DBErrorKeyConflict:        "Такая запись уже есть",
		})
		mod.ConstraintMessages = map[string]string{"items_name_key": "items.errors.name_taken"}
	})
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO public."items"`).WillReturnError(uniqueViolation)
	mock.ExpectRollback()
	w := doDBErrorRequest(engine, http.MethodPut, "/api/items", `{"name":"x"}`)
	assertNoDriverText(t, w, "items_name_key")
	body := decodeDBErrorBody(t, w)
	if w.Code != http.StatusConflict || body.Code != DBErrorCodeConflict || len(body.Errors) != 1 || body.Errors[0] != "Такое имя уже есть" {
		t.Fatalf("unexpected answer %d %+v", w.Code, body)
	}
}

func TestUpdateDeleteViewDoNotLeakDriverError(t *testing.T) {
	engine, mock := newDBErrorRouter(t, true, nil)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE`).WillReturnError(notNullViolation)
	mock.ExpectRollback()
	w := doDBErrorRequest(engine, http.MethodPost, "/api/items/id/1", `{"name":"x"}`)
	assertNoDriverText(t, w, "items")
	if body := decodeDBErrorBody(t, w); w.Code != http.StatusBadRequest || body.Code != DBErrorCodeRequired {
		t.Fatalf("update: %d %+v", w.Code, body)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE`).WillReturnError(foreignKeyViolation)
	mock.ExpectRollback()
	w = doDBErrorRequest(engine, http.MethodDelete, "/api/items/delete/id/1", ``)
	assertNoDriverText(t, w, "orders_item_id_fkey")
	if body := decodeDBErrorBody(t, w); w.Code != http.StatusBadRequest || body.Code != DBErrorCodeReference || body.Message != GeneratorErrorDelete {
		t.Fatalf("delete: %d %+v", w.Code, body)
	}

	mock.ExpectQuery(`SELECT`).WillReturnError(undefinedColumn)
	w = doDBErrorRequest(engine, http.MethodGet, "/api/items/view/id/1", ``)
	assertNoDriverText(t, w, "secret_column")
	if body := decodeDBErrorBody(t, w); w.Code != http.StatusInternalServerError || body.Code != DBErrorCodeInternal || body.Message != dbErrorFallbacks[DBErrorKeyInternal] {
		t.Fatalf("view: %d %+v", w.Code, body)
	}
}

// A hook that asked the database and passed its error on is answered like
// any database error; its own refusal keeps its text.
func TestHookDBErrorIsSanitized(t *testing.T) {
	engine, _ := newDBErrorRouter(t, true, func(gen *Generator, mod *BaseModule) {
		mod.RoleBeforeHook = []actions.RoleHook{{Role: actions.RoleAll, Hook: func(c *gin.Context) error {
			if c.Request.Method == http.MethodGet {
				return actions.NewStatusError(http.StatusForbidden, "Нет доступа")
			}
			return fmt.Errorf("verify: %w", uniqueViolation)
		}}}
	})
	w := doDBErrorRequest(engine, http.MethodPut, "/api/items", `{"name":"x"}`)
	assertNoDriverText(t, w, "items_name_key")
	if w.Code != http.StatusConflict {
		t.Fatalf("hook db error: %d %s", w.Code, w.Body.String())
	}
	w = doDBErrorRequest(engine, http.MethodGet, "/api/items/view/id/1", ``)
	if body := decodeDBErrorBody(t, w); w.Code != http.StatusForbidden || body.Message != "Нет доступа" || body.Code != "" {
		t.Fatalf("hook refusal: %d %+v", w.Code, body)
	}
}

// The reproduction of darkrain/request-generator#167: without a logger in
// the request context a database error panicked into an empty 500.
func TestDBErrorWithoutContextLogger(t *testing.T) {
	for _, withLogger := range []bool{true, false} {
		engine, mock := newDBErrorRouter(t, withLogger, nil)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO public."items"`).WillReturnError(checkViolation)
		mock.ExpectRollback()
		w := doDBErrorRequest(engine, http.MethodPut, "/api/items", `{"name":"x"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("logger in context=%v: expected 400, got %d %q", withLogger, w.Code, w.Body.String())
		}
	}
	if icontext.Logger(context.Background()) == nil {
		t.Fatal("Logger must never be nil")
	}
}
