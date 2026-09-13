package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/grvbrk/nazrein_server/internal/app"
	"github.com/grvbrk/nazrein_server/internal/routes"
)

const (
	PORT string = ":8080"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		slog.Error("Failed to start application", "err", err)
		os.Exit(1)
	}

	slog.SetDefault(app.Logger)

	r, err := routes.SetupRoutes(app)
	if err != nil {
		slog.Error("Failed to set up routes", "err", err)
		os.Exit(1)
	}

	// defer app.RedisClient.Close()

	server := &http.Server{
		Addr:         PORT,
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.Logger.Info("Server started", "port", PORT)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		app.Logger.Error("Error starting server", "err", err)
		os.Exit(1)
	}
}
