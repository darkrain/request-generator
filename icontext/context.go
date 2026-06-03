package icontext

import (
	"context"
	"database/sql"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
)

type key string

const (
	UserContext         = key("userContext")
	LoggerContextKey    = key("loggerContextKey")
	RequestIDContextKey = key("requestIDContextKey")
	DBContextKey        = key("dbContext")
)

func GetContext() context.Context {
	ctx := context.Background()
	guid := xid.New()
	requestID := guid.String()
	requestLogger := log.WithFields(log.Fields{"request_id": requestID})
	ctx = context.WithValue(ctx, LoggerContextKey, requestLogger)
	ctx = context.WithValue(ctx, RequestIDContextKey, requestID)
	return ctx
}

func GetRequestID(ctx context.Context) (*string, bool) {
	u, ok := ctx.Value(RequestIDContextKey).(*string)
	return u, ok
}

// GetLogger - return logger instance from context if it exists.
func GetLogger(ctx context.Context) (*log.Entry, bool) {
	u, ok := ctx.Value(LoggerContextKey).(*log.Entry)
	return u, ok
}

type UserInfo struct {
	ID   int64
	Role string
}

func SetUser(ctx context.Context, user *UserInfo) context.Context {
	return context.WithValue(ctx, UserContext, user)
}

func GetUser(ctx context.Context) (*UserInfo, bool) {
	u, ok := ctx.Value(UserContext).(*UserInfo)
	return u, ok
}

func SetDB(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, DBContextKey, db)
}

func GetDB(ctx context.Context) (*sql.DB, bool) {
	db, ok := ctx.Value(DBContextKey).(*sql.DB)
	return db, ok
}
