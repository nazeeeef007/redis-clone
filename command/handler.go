package command

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nazeeeef007/redis-clone/aof"
	"github.com/nazeeeef007/redis-clone/store"
)

// commandHandler is a function type that defines the signature for all command handling functions.
// All handlers must accept a slice of arguments, the network connection, the data store, and the AOF.
type commandHandler func(args []string, conn net.Conn, s *store.Store, a *aof.AOF)

// Handlers is a map that associates a command name (string) with its corresponding handler function.
// This design makes it easy to add new commands without modifying the core Handle function.
var Handlers = map[string]commandHandler{
	"PING":     ping,
	"SET":      set,
	"GET":      get,
	"DEL":      del,
	"EXISTS":   exists,
	"LPUSH":    lpush,
	"LPOP":     lpop,
	"RPUSH":    rpush,
	"RPOP":     rpop,
	"LRANGE":   lrange,
	"SADD":     sadd,
	"SREM":     srem,
	"SMEMBERS": smembers,
	"HSET":     hset,
	"HGET":     hget,
	"HDEL":     hdel,
	"HGETALL":  hgetall,
}

// Handle routes the incoming command to the correct handler function.
// It checks if the command exists in the Handlers map and executes it.
func Handle(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) == 0 {
		return
	}

	cmd := strings.ToUpper(args[0])
	handler, ok := Handlers[cmd]
	if !ok {
		// If the command is not found, send an unknown command error to the client.
		fmt.Fprintf(conn, "-ERR unknown command '%s'\r\n", cmd)
		return
	}

	// Call the handler function with the command arguments.
	handler(args, conn, s, a)
}

// --- String Commands ---

// ping handles the PING command. It's a simple health check.
func ping(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	fmt.Fprintf(conn, "+PONG\r\n")
}

// set handles the SET command, which stores a string key-value pair.
func set(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'set' command\r\n")
		return
	}
	key := args[1]
	value := args[2]

	// Handle optional TTL arguments (EX for seconds, PX for milliseconds)
	var ttl time.Duration = 0
	if len(args) > 3 {
		option := strings.ToUpper(args[3])
		if option == "EX" && len(args) > 4 {
			seconds, err := strconv.Atoi(args[4])
			if err == nil {
				ttl = time.Duration(seconds) * time.Second
			}
		} else if option == "PX" && len(args) > 4 {
			milliseconds, err := strconv.Atoi(args[4])
			if err == nil {
				ttl = time.Duration(milliseconds) * time.Millisecond
			}
		}
	}

	s.Set(key, value, ttl)
	fmt.Fprintf(conn, "+OK\r\n")

	// Persist the command to the AOF file.
	// This uses a variadic function and the spread operator to pass all elements.
	a.WriteCommand(args[0], args[1:]...)
}

// get handles the GET command, retrieving a string value by its key.
func get(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'get' command\r\n")
		return
	}
	key := args[1]

	val, ok := s.Get(key)
	if !ok {
		fmt.Fprintf(conn, "$-1\r\n") // RESP format for a null bulk string.
		return
	}

	// RESP format for a bulk string.
	fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
}

// del handles the DEL command, removing one or more keys from the store.
func del(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'del' command\r\n")
		return
	}

	count := 0
	for _, key := range args[1:] {
		if s.Del(key) {
			count++
		}
	}
	fmt.Fprintf(conn, ":%d\r\n", count) // RESP integer for the number of deleted keys.
	a.WriteCommand(args[0], args[1:]...)
}

// exists handles the EXISTS command, checking for the existence of one or more keys.
func exists(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'exists' command\r\n")
		return
	}
	count := 0
	for _, key := range args[1:] {
		if s.Exists(key) {
			count++
		}
	}
	fmt.Fprintf(conn, ":%d\r\n", count)
}

// --- List Commands ---

// lpush handles the LPUSH command, adding one or more elements to the head of a list.
func lpush(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'lpush' command\r\n")
		return
	}
	key := args[1]
	elements := args[2:]

	newLen := s.Lpush(key, elements)
	fmt.Fprintf(conn, ":%d\r\n", newLen)

	// Persist the command to the AOF file.
	a.WriteCommand(args[0], args[1:]...)
}

// lpop handles the LPOP command, removing and returning the first element of a list.
func lpop(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'lpop' command\r\n")
		return
	}
	key := args[1]

	val, ok := s.Lpop(key)
	if !ok {
		fmt.Fprintf(conn, "$-1\r\n") // Null bulk string if the list is empty or doesn't exist.
		return
	}

	fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
	a.WriteCommand(args[0], args[1:]...)
}

// rpush handles the RPUSH command, adding one or more elements to the tail of a list.
func rpush(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'rpush' command\r\n")
		return
	}
	key := args[1]
	elements := args[2:]

	newLen := s.Rpush(key, elements)
	fmt.Fprintf(conn, ":%d\r\n", newLen)

	a.WriteCommand(args[0], args[1:]...)
}

// rpop handles the RPOP command, removing and returning the last element of a list.
func rpop(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'rpop' command\r\n")
		return
	}
	key := args[1]

	val, ok := s.Rpop(key)
	if !ok {
		fmt.Fprintf(conn, "$-1\r\n") // Null bulk string if the list is empty or doesn't exist.
		return
	}

	fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
	a.WriteCommand(args[0], args[1:]...)
}

// lrange returns a range of elements from a list.
func lrange(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) != 4 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'lrange' command\r\n")
		return
	}
	key := args[1]

	list := s.Lrange(key)

	start, err1 := strconv.Atoi(args[2])
	end, err2 := strconv.Atoi(args[3])
	if err1 != nil || err2 != nil {
		fmt.Fprintf(conn, "-ERR value is not an integer or out of range\r\n")
		return
	}

	if list == nil {
		fmt.Fprintf(conn, "*0\r\n")
		return
	}

	// Adjust start/end indices for negative values
	if start < 0 {
		start = len(list) + start
	}
	if end < 0 {
		end = len(list) + end
	}

	// Handle out-of-bounds indices
	if start > end || start >= len(list) {
		fmt.Fprintf(conn, "*0\r\n")
		return
	}
	if start < 0 {
		start = 0
	}
	if end >= len(list) {
		end = len(list) - 1
	}

	// Get the sub-slice and return it in RESP array format.
	sublist := list[start : end+1]
	fmt.Fprintf(conn, "*%d\r\n", len(sublist))
	for _, item := range sublist {
		fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(item), item)
	}
}

// --- Set Commands ---

// sadd adds one or more members to a set.
func sadd(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'sadd' command\r\n")
		return
	}
	key := args[1]
	members := args[2:]
	count := s.Sadd(key, members)
	fmt.Fprintf(conn, ":%d\r\n", count)
	a.WriteCommand(args[0], args[1:]...)
}

// srem removes one or more members from a set.
func srem(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'srem' command\r\n")
		return
	}
	key := args[1]
	members := args[2:]
	count := s.Srem(key, members)
	fmt.Fprintf(conn, ":%d\r\n", count)
	a.WriteCommand(args[0], args[1:]...)
}

// smembers returns all members of the set.
func smembers(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'smembers' command\r\n")
		return
	}
	key := args[1]
	members := s.Smembers(key)
	fmt.Fprintf(conn, "*%d\r\n", len(members))
	for _, member := range members {
		fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(member), member)
	}
}

// --- Hash Commands ---

// hset handles the HSET command, which sets a field in a hash.
func hset(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 4 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'hset' command\r\n")
		return
	}
	key := args[1]
	field := args[2]
	value := args[3]
	addedCount := s.HSet(key, field, value)
	fmt.Fprintf(conn, ":%d\r\n", addedCount)
	a.WriteCommand(args[0], args[1:]...)
}

// hget handles the HGET command, which retrieves a value from a hash.
func hget(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'hget' command\r\n")
		return
	}
	key := args[1]
	field := args[2]
	val, ok := s.HGet(key, field)
	if !ok {
		fmt.Fprintf(conn, "$-1\r\n") // RESP format for a null bulk string.
		return
	}
	fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(val), val)
}

// hdel handles the HDEL command, which deletes a field from a hash.
func hdel(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 3 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'hdel' command\r\n")
		return
	}
	key := args[1]
	fields := args[2:]
	deletedCount := s.HDel(key, fields)
	fmt.Fprintf(conn, ":%d\r\n", deletedCount)
	a.WriteCommand(args[0], args[1:]...)
}

// hgetall handles the HGETALL command, which returns all fields and values of a hash.
func hgetall(args []string, conn net.Conn, s *store.Store, a *aof.AOF) {
	if len(args) < 2 {
		fmt.Fprintf(conn, "-ERR wrong number of arguments for 'hgetall' command\r\n")
		return
	}
	key := args[1]
	hash := s.HGetAll(key)
	if hash == nil {
		fmt.Fprintf(conn, "*0\r\n")
		return
	}
	fmt.Fprintf(conn, "*%d\r\n", len(hash)*2)
	for field, value := range hash {
		fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(field), field)
		fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(value), value)
	}
}
