package db

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	sqlitecloud "github.com/sqlitecloud/sqlitecloud-go"
)

var Db *sqlitecloud.SQCloud

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
}

// CloseDB safely closes the database connection
func Close() {
	if Db != nil {
		Db.Close()
		log.Println("Database connection closed")
	}
}
