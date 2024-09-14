package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stathat/consistent"
	"log"
	"net/http"
)

// Struct to hold PostgreSQL clients
type Clients map[string]*pgx.Conn

// Initialize PostgreSQL clients for each shard
var clients Clients

// Consistent hashing ring
var hr *consistent.Consistent

func init() {
	// Initialize consistent hashing ring
	hr = consistent.New()

	// Add ports to the hash ring
	hr.Add("5432")
	hr.Add("5433")
	hr.Add("5434")

	// Initialize PostgreSQL clients
	clients = Clients{
		"5432": connectDB("5432"),
		"5433": connectDB("5433"),
		"5434": connectDB("5434"),
	}
}

// Function to establish connection to PostgreSQL
func connectDB(port string) *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("postgres://postgres:postgres@localhost:%s/postgres", port))
	if err != nil {
		log.Fatalf("Unable to connect to PostgreSQL on port %s: %v\n", port, err)
	}
	return conn
}

func main() {
	// Initialize Gin router
	router := gin.Default()

	// Routes
	router.GET("/:urlId", handleGet)
	router.POST("/", handlePost)

	// Start HTTP server
	log.Println("Listening on port 8088...")
	router.Run(":8088")
}

// GET handler for fetching URL by URL_ID
func handleGet(c *gin.Context) {
	urlId := c.Param("urlId")

	// Get the correct server using consistent hashing
	server, _ := hr.Get(urlId)

	// Execute query on the selected server
	conn := clients[server]
	query := "SELECT * FROM URL_TABLE WHERE URL_ID = $1"
	row := conn.QueryRow(context.Background(), query, urlId)

	// Prepare response
	var url string
	err := row.Scan(&url)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// Send response
	c.JSON(http.StatusOK, gin.H{
		"urlId":  urlId,
		"url":    url,
		"server": server,
	})
}

// POST handler for inserting a new URL
func handlePost(c *gin.Context) {
	// Extract URL from query parameters
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Generate URL_ID using SHA-256 and base64 encoding
	hash := sha256.New()
	hash.Write([]byte(url))
	urlId := base64.StdEncoding.EncodeToString(hash.Sum(nil))[:5]

	// Get the correct server using consistent hashing
	server, _ := hr.Get(urlId)

	// Execute insert query on the selected server
	conn := clients[server]
	query := "INSERT INTO URL_TABLE (URL, URL_ID) VALUES ($1, $2)"
	_, err := conn.Exec(context.Background(), query, url, urlId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert URL"})
		return
	}

	// Send response
	c.JSON(http.StatusOK, gin.H{
		"urlId":  urlId,
		"url":    url,
		"server": server,
	})
}
