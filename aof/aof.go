package aof

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nazeeeef007/redis-clone/store"
)

// AOF represents the Append-Only File. It now includes a mutex for thread-safe operations.
type AOF struct {
	file  *os.File
	store *store.Store
	mu    sync.Mutex
}

// NewAOF creates a new AOF instance and opens the file.
func NewAOF(path string, s *store.Store) (*AOF, error) {
	// Use os.O_RDWR to allow both reading (for Load) and writing (for WriteCommand).
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open AOF file: %w", err)
	}
	return &AOF{file: file, store: s}, nil
}

// WriteCommand appends a command to the AOF file in RESP format.
// This is a significant improvement as it can handle arguments with spaces or special characters.
func (a *AOF) WriteCommand(command string, args ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// RESP format: *<number of arguments>\r\n$<length of arg1>\r\n<arg1>\r\n...
	// We'll write the command and all its arguments as a single RESP array.
	cmdParts := append([]string{command}, args...)
	arrayLen := len(cmdParts)

	// Build the RESP string
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", arrayLen))
	for _, part := range cmdParts {
		b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(part), part))
	}

	_, err := a.file.WriteString(b.String())
	if err != nil {
		return fmt.Errorf("failed to write to AOF: %w", err)
	}
	return nil
}

// Load reads the AOF file and rebuilds the store's state by parsing RESP commands.
func (a *AOF) Load() error {
	log.Println("Loading data from AOF file...")
	file, err := os.OpenFile(a.file.Name(), os.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open AOF file for loading: %w", err)
	}
	defer file.Close()

	// We use a bufio.Reader for more efficient line-by-line reading.
	reader := bufio.NewReader(file)

	for {
		// Read the array length line, e.g., "*3\r\n"
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return fmt.Errorf("error reading AOF array length: %w", err)
		}

		if line[0] != '*' {
			log.Printf("AOF load error: expected array, got %s", line)
			continue
		}

		// Parse the number of arguments.
		arrayLen, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return fmt.Errorf("error parsing AOF array length: %w", err)
		}

		var parts []string
		for i := 0; i < arrayLen; i++ {
			// Read the bulk string length line, e.g., "$5\r\n"
			lenLine, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading AOF bulk string length: %w", err)
			}
			if lenLine[0] != '$' {
				log.Printf("AOF load error: expected bulk string, got %s", lenLine)
				break
			}

			// Parse the length and read the string
			bulkLen, err := strconv.Atoi(strings.TrimSpace(lenLine[1:]))
			if err != nil {
				return fmt.Errorf("error parsing AOF bulk string length: %w", err)
			}

			// Read the actual string data
			data := make([]byte, bulkLen+2) // +2 for "\r\n"
			if _, err := io.ReadFull(reader, data); err != nil {
				return fmt.Errorf("error reading AOF bulk string data: %w", err)
			}

			parts = append(parts, string(data[:bulkLen]))
		}

		// Re-execute the commands to restore the state.
		if len(parts) > 0 {
			command := strings.ToUpper(parts[0])
			args := parts[1:]

			switch command {
			case "SET":
				if len(args) >= 2 {
					a.store.Set(args[0], args[1], 0)
				}
			case "DEL":
				if len(args) >= 1 {
					a.store.Del(args[0])
				}
			case "LPUSH":
				if len(args) >= 2 {
					a.store.Lpush(args[0], args[1:])
				}
			case "RPUSH":
				if len(args) >= 2 {
					a.store.Rpush(args[0], args[1:])
				}
			case "SADD":
				if len(args) >= 2 {
					a.store.Sadd(args[0], args[1:])
				}
			case "SREM":
				if len(args) >= 2 {
					a.store.Srem(args[0], args[1:])
				}
			}
		}
	}

	log.Println("AOF load complete.")
	return nil
}

// Close closes the AOF file.
func (a *AOF) Close() error {
	return a.file.Close()
}
