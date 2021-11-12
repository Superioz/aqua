package main

import (
	"github.com/gin-gonic/gin"
	"github.com/superioz/aqua/internal/middleware"
	"k8s.io/klog"
	"time"
)

func main() {
	klog.Infoln("Hello World!")

	r := gin.New()
	r.Use(middleware.Logger(3 * time.Second))
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	_ = r.Run(":8765")
}
