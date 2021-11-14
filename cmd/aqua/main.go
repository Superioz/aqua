package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/superioz/aqua/internal/handler"
	"github.com/superioz/aqua/internal/middleware"
	"k8s.io/klog"
	"time"
)

// TODO when uploading, if expiration given, change expiration
// TODO Schedule cleanup process for every x minutes and on startup

func main() {
	err := godotenv.Load()
	if err != nil {
		klog.Warningln("Error loading .env file: %v", err)
	}
	klog.Infoln("Hello World!")

	r := gin.New()
	r.Use(middleware.Logger(3 * time.Second))
	// restrict to max 100mb
	r.Use(middleware.RestrictBodySize(100 * handler.SizeMegaByte))
	r.Use(gin.Recovery())

	// handler for receiving uploaded files
	uh := handler.NewUploadHandler()
	r.POST("/upload", uh.Upload)
	err = uh.FileStorage.Cleanup()
	if err != nil {
		klog.Errorln(err)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	_ = r.Run(":8765")
}
