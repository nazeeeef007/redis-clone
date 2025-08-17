// --- File: client/main.go ---
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected to myredis. Type 'quit' to exit.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("myredis> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Exiting.")
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			continue
		}
		line = strings.TrimSpace(line)
		if line == "quit" {
			return
		}

		parts := strings.Split(line, " ")
		cmd := formatRESP(parts)

		// Send command to the server.
		_, err = conn.Write([]byte(cmd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to server: %v\n", err)
			continue
		}

		// Read and display the server's response.
		responseReader := bufio.NewReader(conn)
		resp, err := readRESP(responseReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
			continue
		}
		fmt.Println(resp)
	}
}

// formatRESP converts a slice of strings into a RESP array.
func formatRESP(args []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	return b.String()
}

// readRESP reads and parses a RESP response from the server.
func readRESP(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\r\n")

	switch line[0] {
	case '+': // Simple string
		return line[1:], nil
	case '-': // Error
		return "(error) " + line[1:], nil
	case ':': // Integer
		return line[1:], nil
	case '$': // Bulk string
		length, _ := strconv.Atoi(line[1:])
		if length == -1 {
			return "(nil)", nil
		}
		buf := make([]byte, length)
		_, err = io.ReadFull(r, buf)
		if err != nil {
			return "", err
		}
		r.ReadString('\n') // Read trailing CRLF
		return string(buf), nil
	case '*': // Array
		count, _ := strconv.Atoi(line[1:])
		var result []string
		for i := 0; i < count; i++ {
			item, err := readRESP(r)
			if err != nil {
				return "", err
			}
			result = append(result, item)
		}
		return strings.Join(result, "\n"), nil
	default:
		return "", fmt.Errorf("unexpected RESP response type: %s", line)
	}
}
