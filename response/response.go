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
}

func Response(l *log.Entry, c *gin.Context, data interface{}) {
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
	c.Writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(code)
	response := &responseBase{
		StatusCode: code,
		Message:    message,
		Errors:     data,
	}

	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		l.WithFields(log.Fields{
			"message": message,
		}).Error("Error json encode in Response")
	}
}
