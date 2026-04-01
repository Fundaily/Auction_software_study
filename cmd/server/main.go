package main

import (
	"auction/internal/config"
	"auction/internal/database"
	"auction/internal/handler"
	"auction/internal/middleware"
	"auction/internal/service"
	"auction/internal/ws"
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (overrides AUCTION_CONFIG; default config.yaml)")
	flag.Parse()

	cfg, err := config.LoadWithPath(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}

	hub := ws.NewHub()
	bidLim := middleware.NewBidLimiter(cfg.BidRateBurst, cfg.BidRateEvery)
	deps := handler.Deps{DB: db, Cfg: cfg, BidLim: bidLim, Hub: hub}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	deps.Mount(r)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = service.RefreshAuctionStatuses(db, time.Now())
			}
		}
	}()

	srv := &http.Server{Addr: cfg.Addr, Handler: r}
	go func() {
		log.Printf("listening %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
