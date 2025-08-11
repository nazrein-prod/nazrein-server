package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/grvbrk/nazrein_server/internal/app"
	"github.com/grvbrk/nazrein_server/internal/routes"
	"github.com/joho/godotenv"
)

const (
	PORT string = ":8080"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("PANIC: Error loading env")
		panic(err)
	}

	app, err := app.NewApplication()
	if err != nil {
		app.Logger.Fatal("Error creating new Application", err)
	}

	r := routes.SetupRoutes(app)

	// defer app.RedisClient.Close()

	server := &http.Server{
		Addr:         PORT,
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.Logger.Println("Server started on port", PORT)

	err = server.ListenAndServe()
	if err != nil {
		app.Logger.Fatal("Error starting server", err)
	}

}
