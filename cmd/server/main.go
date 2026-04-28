package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Kerem451I/uptime-monitor/internal/api"
	"github.com/Kerem451I/uptime-monitor/internal/checker"
	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	// after godotenv.Load(), the program can access DATABASE_URL as an environment variable.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	required := map[string]string{
		"DB_USER":     dbUser,
		"DB_PASSWORD": dbPassword,
		"DB_NAME":     dbName,
		"DB_HOST":     dbHost,
	}

	for key, val := range required {
		if val == "" {
			log.Fatalf("%s must be set", key)
		}
	}

	if dbPort == "" {
		dbPort = "5432" // default
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	pool, err := db.New(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("connected to database successfully")

	runMigrations(connString)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// create a context that listens for Ctrl+C (SIGINT) or SIGTERM
	// this replaces context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// context for lifecycle
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// initialize the logic/handlers
	chkr := checker.New(pool)
	limiter := api.NewIPRateLimiter(10, 20)

	handler := api.NewHandler(pool)
	router := api.NewRouter(handler, limiter)

	// start background goroutines
	go chkr.Run(ctx)
	go limiter.Start(ctx)

	// define the server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// start the server in a goroutine
	go func() {
		log.Printf("server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// waits here until the context is cancelled
	<-ctx.Done()

	log.Println("shutting down gracefully...")

	// give the server 5 seconds to finish active (in flight) requests
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited.")

}

func runMigrations(connString string) {
	pgxConnString := strings.Replace(connString, "postgres://", "pgx5://", 1)
	m, err := migrate.New("file://migrations", pgxConnString)
	if err != nil {
		log.Fatalf("could not create migrate instance: %v", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("migrations: nothing to apply")
			return
		}
		log.Fatalf("could not run migrations: %v", err)
	}

	log.Println("migrations: applied successfully")
}
