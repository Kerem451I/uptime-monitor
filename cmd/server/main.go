package main

import (
	"log"
	"os"

	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}
	// after godotenv.Load(), the program can access DATABASE_URL as an environment variable.

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := db.New(connString)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("connected to database successfully")
}
