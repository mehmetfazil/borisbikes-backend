package db

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	sqlitecloud "github.com/sqlitecloud/sqlitecloud-go"
)

var (
	Db              *sqlitecloud.SQCloud
	keepAliveTicker *time.Ticker
	quit            chan struct{}
)

func Init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	connStr := os.Getenv("SQLITECLOUD_CONN_STR")
	if connStr == "" {
		log.Fatal("SQLITECLOUD_CONN_STR environment variable not set")
	}

	var err error
	Db, err = sqlitecloud.Connect(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite Cloud: %v", err)
	}
	log.Println("Successfully connected to SQLite Cloud")

	// Start the keep-alive routine
	startKeepAlive(5 * time.Minute) // Adjust the interval as needed
}

// startKeepAlive starts a goroutine that periodically checks the database connection.
func startKeepAlive(interval time.Duration) {
	keepAliveTicker = time.NewTicker(interval)
	quit = make(chan struct{})

	go func() {
		for {
			select {
			case <-keepAliveTicker.C:
				if _, err := Db.Select("SELECT 1;"); err != nil {
					log.Printf("Keep-alive query failed: %v", err)
					log.Println("Attempting to reconnect to SQLite Cloud...")
					Db.Close()
					connStr := os.Getenv("SQLITECLOUD_CONN_STR")
					Db, err = sqlitecloud.Connect(connStr)
					if err != nil {
						log.Printf("Reconnection attempt failed: %v", err)
					} else {
						log.Println("Reconnected to SQLite Cloud")
					}
				} else {
					log.Println("Keep-alive query succeeded")
				}
			case <-quit:
				return
			}
		}
	}()
}

func Close() {
	if Db != nil {
		Db.Close()
		log.Println("Database connection closed")
	}
	if keepAliveTicker != nil {
		keepAliveTicker.Stop()
	}
	if quit != nil {
		close(quit)
	}
}
