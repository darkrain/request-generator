package response

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type responseBase struct {
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Errors     interface{} `json:"errors"`
	// Code is a stable name of the refusal (for example "conflict") when
	// the library classified it; a client may branch on it.
	Code string `json:"code,omitempty"`
}

func Response(l *log.Entry, c *gin.Context, data interface{}) {
	if l == nil {
		l = log.NewEntry(log.StandardLogger())
	}
	c.Writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(http.StatusOK)
	response := data

	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		l.WithFields(log.Fields{
			"Data": data,
		}).Error("Error json encode in Response")
	}
}

func ResponseCSV(l *log.Entry, c *gin.Context, data []byte) {
	if l == nil {
		l = log.NewEntry(log.StandardLogger())
	}
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=\"csvResult.csv\"")
	c.Writer.WriteHeader(http.StatusOK)

	if _, err := c.Writer.Write(data); err != nil {
		l.WithFields(log.Fields{
			"Data": data,
		}).Error("Error json encode in Response")
	}
}

// ErrorResponse generate a error response
func ErrorResponse(l *log.Entry, c *gin.Context, code int, message string, data interface{}) {
	ErrorResponseWithCode(l, c, code, message, data, "")
}

// ErrorResponseWithCode is ErrorResponse with a stable error code.
func ErrorResponseWithCode(l *log.Entry, c *gin.Context, code int, message string, data interface{}, errorCode string) {
	if l == nil {
		l = log.NewEntry(log.StandardLogger())
	}
	c.Writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(code)
	response := &responseBase{
		StatusCode: code,
		Message:    message,
		Errors:     data,
		Code:       errorCode,
	}

	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		l.WithFields(log.Fields{
			"message": message,
		}).Error("Error json encode in Response")
	}
}
