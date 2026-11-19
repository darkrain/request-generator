package module

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/response"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Codes a database refusal answers with (`code` in the error response). They
// are stable: a client may rely on them, unlike on the driver's wording.
const (
	DBErrorCodeConflict     = "conflict"
	DBErrorCodeReference    = "reference"
	DBErrorCodeInvalidValue = "invalid_value"
	DBErrorCodeRequired     = "required"
	DBErrorCodeNotFound     = "not_found"
	DBErrorCodeInternal     = "internal"
)

// Translation keys of the generic texts. An application translates them in
// its locale files; the English fallback is used when it does not.
const (
	DBErrorKeyConflict     = "generator.errors.db.conflict"
	DBErrorKeyReference    = "generator.errors.db.reference"
	DBErrorKeyInvalidValue = "generator.errors.db.invalid_value"
	DBErrorKeyRequired     = "generator.errors.db.required"
	DBErrorKeyNotFound     = "generator.errors.db.not_found"
	DBErrorKeyInternal     = "generator.errors.db.internal"
)

var dbErrorFallbacks = map[string]string{
	DBErrorKeyConflict:     "This record already exists.",
	DBErrorKeyReference:    "The record refers to data that does not exist or is still in use.",
	DBErrorKeyInvalidValue: "One of the values is not allowed.",
	DBErrorKeyRequired:     "A required value is missing.",
	DBErrorKeyNotFound:     "Record not found.",
	DBErrorKeyInternal:     "Something went wrong. Please try again later.",
}

// DBError is a database failure as a client may see it: a status, a stable
// code and a translation key. The driver's own text - with table, column and
// constraint names - stays in Err and in the server log.
type DBError struct {
	Status     int
	Code       string
	Key        string
	SQLState   string
	Constraint string
	Err        error
}

func (e *DBError) Error() string { return e.Err.Error() }
func (e *DBError) Unwrap() error { return e.Err }

// sqlStater is what both lib/pq (*pq.Error) and pgx (*pgconn.PgError) errors
// answer, so neither driver is a dependency of the library.
type sqlStater interface{ SQLState() string }

// driverTextPrefixes are the wordings of database errors that carry no
// SQLSTATE: lib/pq protocol errors, database/sql scan errors and go-jet's
// wrappers. They are never shown to a client either.
var driverTextPrefixes = []string{"pq: ", "sql: ", "jet: ", "qrm: ", "pgx: ", "pgconn: ", "driver: "}

// ClassifyDBError tells whether err comes from the database and, if so, how a
// client is answered. Errors the library or an application makes itself -
// validation messages, StatusError refusals of hooks - are not database
// errors and keep their text.
func ClassifyDBError(err error) (*DBError, bool) {
	if err == nil {
		return nil, false
	}
	var classified *DBError
	if errors.As(err, &classified) {
		return classified, true
	}
	var stater sqlStater
	if errors.As(err, &stater) {
		state := stater.SQLState()
		out := &DBError{SQLState: state, Constraint: constraintName(stater), Err: err}
		switch {
		case state == "23505" || state == "23P01":
			out.Status, out.Code, out.Key = http.StatusConflict, DBErrorCodeConflict, DBErrorKeyConflict
		case state == "23503" || state == "23001":
			out.Status, out.Code, out.Key = http.StatusBadRequest, DBErrorCodeReference, DBErrorKeyReference
		case state == "23502":
			out.Status, out.Code, out.Key = http.StatusBadRequest, DBErrorCodeRequired, DBErrorKeyRequired
		case state == "23514" || strings.HasPrefix(state, "22") || strings.HasPrefix(state, "23"):
			// Integrity and data exceptions are a value the database refused:
			// a check, a malformed uuid or number, a string too long.
			out.Status, out.Code, out.Key = http.StatusBadRequest, DBErrorCodeInvalidValue, DBErrorKeyInvalidValue
		default:
			out.Status, out.Code, out.Key = http.StatusInternalServerError, DBErrorCodeInternal, DBErrorKeyInternal
		}
		return out, true
	}
	if errors.Is(err, sql.ErrNoRows) {
		return &DBError{Status: http.StatusNotFound, Code: DBErrorCodeNotFound, Key: DBErrorKeyNotFound, Err: err}, true
	}
	var netErr net.Error
	// A cancelled or timed-out request keeps its text: it names no part of
	// the database, and the client that went away reads nothing anyway.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) || errors.Is(err, sql.ErrTxDone) || errors.As(err, &netErr) {
		return &DBError{Status: http.StatusInternalServerError, Code: DBErrorCodeInternal, Key: DBErrorKeyInternal, Err: err}, true
	}
	text := err.Error()
	if strings.Contains(text, "SQLSTATE") {
		return &DBError{Status: http.StatusInternalServerError, Code: DBErrorCodeInternal, Key: DBErrorKeyInternal, Err: err}, true
	}
	for _, prefix := range driverTextPrefixes {
		if strings.HasPrefix(text, prefix) || strings.Contains(text, ": "+prefix) {
			if strings.Contains(text, "no rows in result set") {
				return &DBError{Status: http.StatusNotFound, Code: DBErrorCodeNotFound, Key: DBErrorKeyNotFound, Err: err}, true
			}
			return &DBError{Status: http.StatusInternalServerError, Code: DBErrorCodeInternal, Key: DBErrorKeyInternal, Err: err}, true
		}
	}
	return nil, false
}

// constraintName reads the constraint of a driver error: Constraint on
// *pq.Error, ConstraintName on *pgconn.PgError.
func constraintName(err interface{}) string {
	value := reflect.ValueOf(err)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range []string{"Constraint", "ConstraintName"} {
		if field := value.FieldByName(name); field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}
	return ""
}

// clientError is what a client is told about err. A database error is logged
// in full and answered with a localized generic text and a stable code; a
// module names its own text for a constraint in BaseModule.ConstraintMessages
// (a translation key per constraint name). Any other error keeps its status
// and text.
func (generator *Generator) clientError(c *gin.Context, l *log.Entry, module *BaseModule, status int, err error) (int, string, string) {
	if err == nil {
		return status, "", ""
	}
	classified, ok := ClassifyDBError(err)
	if !ok {
		return status, err.Error(), ""
	}
	if l == nil {
		l = icontext.Logger(c.Request.Context())
	}
	fields := log.Fields{"sqlstate": classified.SQLState, "code": classified.Code}
	if classified.Constraint != "" {
		fields["constraint"] = classified.Constraint
	}
	if module != nil {
		fields["module"] = module.Name
	}
	l.WithFields(fields).Errorln("DB ERR: ", classified.Err)
	lang := generator.DefaultLocale
	if value, ok := c.Get(translationLangContextKey); ok {
		if current, ok := value.(locale.Lang); ok {
			lang = current
		}
	}
	return classified.Status, generator.dbErrorText(lang, module, classified), classified.Code
}

func (generator *Generator) dbErrorText(lang locale.Lang, module *BaseModule, classified *DBError) string {
	fallback := dbErrorFallbacks[classified.Key]
	if fallback == "" {
		fallback = dbErrorFallbacks[DBErrorKeyInternal]
	}
	text := generator.TranslateWithFallback(lang, classified.Key, fallback)
	if classified.Constraint == "" {
		return text
	}
	for _, messages := range []map[string]string{moduleConstraintMessages(module), generator.ConstraintMessages} {
		if key := messages[classified.Constraint]; key != "" {
			return generator.TranslateWithFallback(lang, key, text)
		}
	}
	return text
}

func moduleConstraintMessages(module *BaseModule) map[string]string {
	if module == nil {
		return nil
	}
	return module.ConstraintMessages
}

// failMessage answers with the error as the message.
func (generator *Generator) failMessage(c *gin.Context, l *log.Entry, module *BaseModule, status int, err error) {
	status, text, code := generator.clientError(c, l, module, status, err)
	response.ErrorResponseWithCode(l, c, status, text, nil, code)
}

// failErrors answers with message and the error in errors.
func (generator *Generator) failErrors(c *gin.Context, l *log.Entry, module *BaseModule, status int, message string, err error) {
	status, text, code := generator.clientError(c, l, module, status, err)
	response.ErrorResponseWithCode(l, c, status, message, []string{text}, code)
}

// failBoth answers with the error as the message and in errors.
func (generator *Generator) failBoth(c *gin.Context, l *log.Entry, module *BaseModule, status int, err error) {
	status, text, code := generator.clientError(c, l, module, status, err)
	response.ErrorResponseWithCode(l, c, status, text, []string{text}, code)
}
