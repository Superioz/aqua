package middleware

import (
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
	"time"
)

func setupLogging(duration time.Duration) {
	go func() {
		for range time.Tick(duration) {
			klog.Flush()
		}
	}()
}

// Logger is the middleware to print the Gin logging
// to our own logger. That way the logging messages are more
// streamlined and easier to maintain.
//
// Taken from https://github.com/szuecs/gin-glog
func Logger(duration time.Duration) gin.HandlerFunc {
	setupLogging(duration)
	return func(c *gin.Context) {
		t := time.Now()

		// process request
		c.Next()

		latency := time.Since(t)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		path := c.Request.URL.Path

		switch {
		case statusCode >= 400 && statusCode <= 499:
			{
				klog.Warningf("[GIN] | %3d | %12v | %s | %-7s %s\n%s",
					statusCode,
					latency,
					clientIP,
					method,
					path,
					c.Errors.String(),
				)
			}
		case statusCode >= 500:
			{
				klog.Errorf("[GIN] | %3d | %12v | %s | %-7s %s\n%s",
					statusCode,
					latency,
					clientIP,
					method,
					path,
					c.Errors.String(),
				)
			}
		default:
			klog.Infof("[GIN] | %3d | %12v | %s | %-7s %s\n%s",
				statusCode,
				latency,
				clientIP,
				method,
				path,
				c.Errors.String(),
			)
		}

	}
}
