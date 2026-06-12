package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.GET("/api/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	log.Printf("server starting on :8080 (env=%s)", cfg.AppEnv)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server: %v", err)
	}
}
