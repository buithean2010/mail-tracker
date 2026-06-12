package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/config"
	"github.com/buithean2010/mail-tracker/backend/internal/handler"
	infrai "github.com/buithean2010/mail-tracker/backend/internal/infra/ai"
	"github.com/buithean2010/mail-tracker/backend/internal/infra/crypto"
	"github.com/buithean2010/mail-tracker/backend/internal/infra/db"
	"github.com/buithean2010/mail-tracker/backend/internal/infra/graph"
	"github.com/buithean2010/mail-tracker/backend/internal/infra/sse"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	pool, err := db.Connect(context.Background(), cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	enc, err := crypto.NewAES(cfg.AESKey)
	if err != nil {
		log.Fatalf("crypto: %v", err)
	}

	userRepo := db.NewUserRepo(pool)
	sessionRepo := db.NewSessionRepo(pool)
	tokenRepo := db.NewTokenRepo(pool, enc)
	threadRepo := db.NewThreadRepo(pool)
	apiKeyRepo := db.NewAPIKeyRepo(pool, enc)

	graphClient := graph.NewClient()
	openaiClient := infrai.NewOpenAIClient()
	openrouterClient := infrai.NewOpenRouterClient()
	paClient := infrai.NewPAClient()

	broker := sse.NewBroker()
	stopBroker := make(chan struct{})
	go broker.Heartbeat(stopBroker)

	authUC := app.NewAuthUseCase(cfg, userRepo, sessionRepo, tokenRepo)
	syncUC := app.NewSyncUseCase(userRepo, sessionRepo, tokenRepo, threadRepo, graphClient, broker, enc, authUC)
	summaryUC := app.NewSummaryUseCase(threadRepo, apiKeyRepo, openaiClient, openrouterClient, paClient, enc)
	threadUC := app.NewThreadUseCase(threadRepo)
	userUC := app.NewUserUseCase(userRepo, apiKeyRepo, enc)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	handler.RegisterRoutes(r, handler.Deps{
		AuthUC:      authUC,
		SyncUC:      syncUC,
		SummaryUC:   summaryUC,
		ThreadUC:    threadUC,
		UserUC:      userUC,
		SessionRepo: sessionRepo,
		UserRepo:    userRepo,
		TokenRepo:   tokenRepo,
		Broker:      broker,
		Enc:         enc,
		Secure:      cfg.AppEnv == "production",
		FrontendURL: cfg.BaseURL,
	})

	go startJobs(cfg, syncUC, summaryUC)

	srv := &http.Server{
		Addr:        ":8080",
		Handler:     r,
		ReadTimeout: 30 * time.Second,
		// WriteTimeout intentionally unset — SSE needs long-lived connections
	}

	log.Printf("server starting on :8080 (env=%s)", cfg.AppEnv)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	close(stopBroker)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func startJobs(cfg *config.Config, syncUC *app.SyncUseCase, summaryUC *app.SummaryUseCase) {
	interval := time.Duration(cfg.SyncIntervalMinutes) * time.Minute
	runJobs(syncUC, summaryUC)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		runJobs(syncUC, summaryUC)
	}
}

func runJobs(syncUC *app.SyncUseCase, summaryUC *app.SummaryUseCase) {
	ctx := context.Background()

	log.Println("[job1] mail sync start")
	if err := syncUC.SyncAll(ctx); err != nil {
		log.Printf("[job1] error: %v", err)
	}
	log.Println("[job1] done")

	log.Println("[job2] AI summary start")
	if err := summaryUC.RunAll(ctx); err != nil {
		log.Printf("[job2] error: %v", err)
	}
	log.Println("[job2] done")
}
