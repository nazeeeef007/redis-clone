// resp.go
package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// The different types of RESP messages.
const (
	SimpleString = '+'
	Error        = '-'
	Integer      = ':'
	BulkString   = '$'
	Array        = '*'
)

// Value represents a generic RESP value.
type Value struct {
	Type    byte
	String  string
	Array   []Value
	Integer int // Added a field to store integer values.
}

// RESP is a parser and serializer for the Redis Serialization Protocol.
// It holds both a reader and a writer to handle bidirectional communication.
type RESP struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewRESP creates a new RESP parser instance.
func NewRESP(rw io.ReadWriter) *RESP {
	return &RESP{
		reader: bufio.NewReader(rw),
		writer: bufio.NewWriter(rw),
	}
}

// ReadArray reads and parses a RESP Array message, which is the typical format
// for client commands.
func (r *RESP) ReadArray() ([]string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line[0] != Array {
		return nil, fmt.Errorf("invalid RESP format: expected array start, got '%c'", line[0])
	}

	num, err := strconv.Atoi(line[1 : len(line)-2])
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %w", err)
	}
	if num == -1 {
		return nil, nil
	}

	args := make([]string, num)
	for i := 0; i < num; i++ {
		val, err := r.ReadBulkString()
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	return args, nil
}

// ReadBulkString reads and parses a RESP Bulk String.
func (r *RESP) ReadBulkString() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if line[0] != BulkString {
		return "", fmt.Errorf("invalid RESP format: expected bulk string, got '%c'", line[0])
	}

	length, err := strconv.Atoi(line[1 : len(line)-2])
	if err != nil {
		return "", fmt.Errorf("invalid bulk string length: %w", err)
	}
	if length == -1 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r.reader, buf); err != nil {
		return "", err
	}

	if _, err := r.reader.ReadString('\n'); err != nil {
		return "", err
	}

	return string(buf), nil
}

// WriteString writes a simple string response.
func (r *RESP) WriteString(s string) error {
	_, err := r.writer.WriteString(fmt.Sprintf("+%s\r\n", s))
	if err != nil {
		return err
	}
	return r.writer.Flush()
}

// WriteError writes an error response.
func (r *RESP) WriteError(s string) error {
	_, err := r.writer.WriteString(fmt.Sprintf("-%s\r\n", s))
	if err != nil {
		return err
	}
	return r.writer.Flush()
}

// WriteInteger writes an integer response.
func (r *RESP) WriteInteger(i int) error {
	_, err := r.writer.WriteString(fmt.Sprintf(":%d\r\n", i))
	if err != nil {
		return err
	}
	return r.writer.Flush()
}

// WriteBulkString writes a bulk string response.
func (r *RESP) WriteBulkString(s string) error {
	_, err := r.writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
	if err != nil {
		return err
	}
	return r.writer.Flush()
}

// WriteNull writes a null response.
func (r *RESP) WriteNull() error {
	_, err := r.writer.WriteString("$-1\r\n")
	if err != nil {
		return err
	}
	return r.writer.Flush()
}

// WriteArray writes a RESP array response.
func (r *RESP) WriteArray(vals []Value) error {
	_, err := r.writer.WriteString(fmt.Sprintf("*%d\r\n", len(vals)))
	if err != nil {
		return err
	}
	for _, val := range vals {
		if err := r.WriteValue(val); err != nil {
			return err
		}
	}
	return r.writer.Flush()
}

// WriteValue writes a single RESP value.
func (r *RESP) WriteValue(v Value) error {
	switch v.Type {
	case SimpleString:
		return r.WriteString(v.String)
	case Error:
		return r.WriteError(v.String)
	case Integer:
		return r.WriteInteger(v.Integer)
	case BulkString:
		return r.WriteBulkString(v.String)
	case Array:
		return r.WriteArray(v.Array)
	}
	return nil
}
