package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"references/internal/config"
	"references/internal/game"
	"references/internal/handlers"
)

func main() {
	cfg := config.Load()

	g, err := game.NewGame(cfg)
	if err != nil {
		log.Fatalf("initialise game: %v", err)
	}

	h := handlers.NewHandlers(g)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.IndexHandler)
	mux.HandleFunc("/guess", h.GuessHandler)
	mux.HandleFunc("/hint", h.HintHandler)
	mux.HandleFunc("/stats", h.StatsHandler)
	mux.HandleFunc("/success", h.SuccessHandler)
	mux.HandleFunc("/maybe-tomorrow", h.MaybeTomorrowHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s (mode=%s)", srv.Addr, cfg.Mode)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal: %v", sig)
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
