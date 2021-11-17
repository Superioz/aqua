package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"github.com/superioz/aqua/internal/handler"
	"github.com/superioz/aqua/internal/storage"
	"github.com/superioz/aqua/pkg/env"
	"github.com/superioz/aqua/pkg/middleware"
	"k8s.io/klog"
	"time"
)

// TODO add structure diagram (api for uploading, file backend with metadata database, scheduler for expiration)

func main() {
	err := godotenv.Load()
	if err != nil {
		klog.Warningf("Error loading .env file: %v", err)
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
	_, err = s.Every(env.IntOrDefault("FILE_EXPIRATION_CYCLE", 15)).Minutes().StartImmediately().Do(func() {
		err = uh.FileStorage.Cleanup()
		if err != nil {
			klog.Errorln(err)
		}
	})
	if err != nil {
		klog.Fatalf("could not start cleanup scheduler: %v", err)
	}
	s.StartAsync()

	r.Static("/", env.StringOrDefault("FILE_STORAGE_PATH", storage.EnvDefaultFileStoragePath))

	_ = r.Run(":8765")
}
