// --- File: main.go ---
package main

import (
	"log"

	"github.com/nazeeeef007/redis-clone/server"
)

func main() {
	// Create a new server instance.
	srv := server.NewServer()

	// Listen and serve on port 6379, the default Redis port.
	log.Println("Starting myredis server on :6379...")
	if err := srv.Listen(":6379"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
