package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/nazeeeef007/redis-clone/aof"
	"github.com/nazeeeef007/redis-clone/command"
	"github.com/nazeeeef007/redis-clone/resp"
	"github.com/nazeeeef007/redis-clone/store"
)

// Server holds the state of our Redis clone.
type Server struct {
	store *store.Store
	aof   *aof.AOF
	mu    sync.RWMutex
}

// NewServer creates a new Server instance.
func NewServer() *Server {
	s := &Server{
		store: store.NewStore(),
	}

	// Initialize and load the AOF.
	var err error
	s.aof, err = aof.NewAOF("myredis.aof", s.store)
	if err != nil {
		log.Fatalf("Failed to initialize AOF: %v", err)
	}
	if err := s.aof.Load(); err != nil {
		log.Fatalf("Failed to load AOF: %v", err)
	}

	return s
}

// Listen starts the TCP server on the given address.
func (s *Server) Listen(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("myredis server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		// Handle each connection in a new goroutine.
		go s.handleConnection(conn)
	}
}

// handleConnection manages a single client connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("New client connected: %s", conn.RemoteAddr())

	// Create a new RESP parser for this connection.
	parser := resp.NewRESP(conn)

	for {
		// Read RESP command from the client. The parser handles the entire command.
		args, err := parser.ReadArray()
		if err != nil {
			if err == io.EOF {
				log.Printf("Client disconnected: %s", conn.RemoteAddr())
			} else {
				log.Printf("RESP parse error: %v", err)
				conn.Write([]byte(fmt.Sprintf("-(error) %v\r\n", err)))
			}
			return
		}

		// Lock the server's data for thread-safe access.
		s.mu.Lock()

		// Use the new command handler to process the request.
		command.Handle(args, conn, s.store, s.aof)

		// Unlock when done.
		s.mu.Unlock()
	}
}
