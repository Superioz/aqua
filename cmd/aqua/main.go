package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/superioz/aqua/internal/handler"
	"github.com/superioz/aqua/internal/middleware"
	"k8s.io/klog"
	"time"
)

// TODO Add cleanup process, to delete all images that are not in the sqlite or that are expired
// TODO make file storage (not metadata) an adapter (AddFile, RemoveFile)

func main() {
	err := godotenv.Load()
	if err != nil {
		klog.Warningln("Error loading .env file: %v", err)
	}
	klog.Infoln("Hello World!")

	// init some stuff for the handler
	// like config etc.
	handler.Initialize()

	r := gin.New()
	r.Use(middleware.Logger(3 * time.Second))
	// restrict to max 100mb
	r.Use(middleware.RestrictBodySize(100 * handler.SizeMegaByte))
	r.Use(gin.Recovery())

	r.POST("/upload", handler.Upload)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	_ = r.Run(":8765")
}
