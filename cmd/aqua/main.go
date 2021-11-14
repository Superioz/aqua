package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"github.com/superioz/aqua/internal/handler"
	"github.com/superioz/aqua/internal/middleware"
	"github.com/superioz/aqua/pkg/env"
	"k8s.io/klog"
	"time"
)

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

	// scheduler to do the cleanup every x minutes
	s := gocron.NewScheduler(time.UTC)
	s.Every(env.IntOrDefault("FILE_EXPIRATION_CYCLE", 15)).Minutes().StartImmediately().Do(func() {
		err = uh.FileStorage.Cleanup()
		if err != nil {
			klog.Errorln(err)
		}
	})
	s.StartAsync()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})
	_ = r.Run(":8765")
}
