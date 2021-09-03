package gcplog

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

func Gin(projectId string, serviceName string, resource string) gin.HandlerFunc {
	Init(projectId, serviceName)
	return func(c *gin.Context) {

		gcplog := NewGcpLog(resource)
		defer gcplog.Close()

		// before request
		// log the body maybe..
		// ...do something

		defer func(begin time.Time) {

			// after request
			status := c.Writer.Status()
			log := c.Request.Method + " " + c.Request.URL.Path

			request := &logging.HTTPRequest{
				Request:      c.Request,
				RequestSize:  c.Request.ContentLength,
				Status:       c.Writer.Status(),
				ResponseSize: int64(c.Writer.Size()),
				Latency:      time.Since(begin),

				LocalIP:  c.ClientIP(),
				RemoteIP: c.Request.RemoteAddr,
			}
			var trace string
			traceHeader := c.Request.Header.Get("X-Cloud-Trace-Context")
			traceParts := strings.Split(traceHeader, "/")
			if len(traceParts) > 0 && len(traceParts[0]) > 0 {
				trace = fmt.Sprintf("projects/%s/traces/%s", projectId, traceParts[0])
			}

			if status < 400 {
				gcplog.Log(LogEntry{
					log:     log,
					trace:   trace,
					request: request,
				})
				return
			}

			var err error
			if len(c.Errors) > 0 {
				err = c.Errors.Last().Err
			} else {
				err = fmt.Errorf(log)
			}

			if status >= 400 && status < 500 {
				gcplog.Warn(ErrorEntry{
					err:     err,
					trace:   trace,
					request: request,
				})
			} else {
				gcplog.Error(ErrorEntry{
					err:     err,
					trace:   trace,
					request: request,
				})
			}
		}(time.Now())

		c.Next()
	}
}