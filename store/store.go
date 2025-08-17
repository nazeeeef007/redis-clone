// store/store.go
package store

import (
	"log"
	"sync"
	"time"
)

// The different types of data we support.
type DataType int

const (
	TypeString DataType = iota
	TypeList
	TypeSet
	TypeHash // A hash map from string fields to string values.
)

// Item holds the value and optional expiration time.
type Item struct {
	Value      interface{}
	Type       DataType
	Expiration time.Time
}

// Store is our in-memory data store. It now uses a slice of RWMutexes for fine-grained locking.
type Store struct {
	items map[string]Item
	// locks is a slice of read-write mutexes used to protect individual keys.
	// Using a fixed size prevents an unbounded number of mutexes.
	locks []sync.RWMutex
}

// NewStore creates a new Store instance. It initializes the map and the array of locks.
func NewStore() *Store {
	const numLocks = 256 // A common practice, provides a good balance between memory and contention.
	locks := make([]sync.RWMutex, numLocks)

	s := &Store{
		items: make(map[string]Item),
		locks: locks,
	}

	// Start the background worker for active expiration.
	go s.activeExpirationWorker()
	return s
}

// getLock returns the correct RWMutex for a given key by hashing the key.
// This ensures that all operations on a specific key use the same lock.
func (s *Store) getLock(key string) *sync.RWMutex {
	// Simple non-cryptographic hash for performance.
	var hash uint32
	for _, char := range key {
		hash = 31*hash + uint32(char)
	}
	return &s.locks[hash%uint32(len(s.locks))]
}

// isExpired checks if an item has expired. This function
// is for internal use and does NOT handle locking.
func (s *Store) isExpired(item Item) bool {
	return !item.Expiration.IsZero() && time.Now().After(item.Expiration)
}

// Set sets a key-value pair with an optional time-to-live (TTL).
func (s *Store) Set(key string, value string, ttl time.Duration) {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	s.items[key] = Item{
		Value:      value,
		Type:       TypeString,
		Expiration: expiration,
	}
}

// Get retrieves a value for a given key, performing passive expiration.
func (s *Store) Get(key string) (string, bool) {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok {
		return "", false
	}

	if s.isExpired(item) {
		s.Del(key) // This call to Del handles its own locking.
		return "", false
	}

	strVal, ok := item.Value.(string)
	if !ok || item.Type != TypeString {
		return "", false // Key exists but is of the wrong type.
	}
	return strVal, true
}

// Del deletes a key from the store.
func (s *Store) Del(key string) bool {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()
	if _, ok := s.items[key]; ok {
		delete(s.items, key)
		return true
	}
	return false
}

// Exists checks if a key exists and has not expired.
func (s *Store) Exists(key string) bool {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok {
		return false
	}

	if s.isExpired(item) {
		s.Del(key)
		return false
	}

	return true
}

// Lpush adds elements to the beginning of a list.
func (s *Store) Lpush(key string, values []string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	var list []string
	if ok {
		if item.Type != TypeList {
			delete(s.items, key)
			list = []string{}
		} else {
			list = item.Value.([]string)
		}
	} else {
		list = []string{}
	}

	newlist := make([]string, len(values)+len(list))
	copy(newlist, values)
	copy(newlist[len(values):], list)
	s.items[key] = Item{Value: newlist, Type: TypeList, Expiration: item.Expiration}
	return len(newlist)
}

// Rpush adds elements to the end of a list.
func (s *Store) Rpush(key string, values []string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	var list []string
	if ok {
		if item.Type != TypeList {
			delete(s.items, key)
			list = []string{}
		} else {
			list = item.Value.([]string)
		}
	} else {
		list = []string{}
	}
	newlist := append(list, values...)
	s.items[key] = Item{Value: newlist, Type: TypeList, Expiration: item.Expiration}
	return len(newlist)
}

// Lpop removes and returns the first element of a list.
func (s *Store) Lpop(key string) (string, bool) {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeList || s.isExpired(item) {
		return "", false
	}

	list := item.Value.([]string)
	if len(list) == 0 {
		return "", false
	}
	val := list[0]
	if len(list[1:]) == 0 {
		delete(s.items, key)
	} else {
		s.items[key] = Item{Value: list[1:], Type: TypeList, Expiration: item.Expiration}
	}
	return val, true
}

// Rpop removes and returns the last element of a list.
func (s *Store) Rpop(key string) (string, bool) {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeList || s.isExpired(item) {
		return "", false
	}

	list := item.Value.([]string)
	if len(list) == 0 {
		return "", false
	}
	val := list[len(list)-1]
	if len(list[:len(list)-1]) == 0 {
		delete(s.items, key)
	} else {
		s.items[key] = Item{Value: list[:len(list)-1], Type: TypeList, Expiration: item.Expiration}
	}
	return val, true
}

// Llen returns the length of a list.
func (s *Store) Llen(key string) int {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok || item.Type != TypeList || s.isExpired(item) {
		return 0
	}
	list := item.Value.([]string)
	return len(list)
}

// Lrange returns a slice of a list. For simplicity, we return the whole list.
func (s *Store) Lrange(key string) []string {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok || item.Type != TypeList || s.isExpired(item) {
		return nil
	}
	// Return a copy to prevent external modifications.
	list := item.Value.([]string)
	newList := make([]string, len(list))
	copy(newList, list)
	return newList
}

// Sadd adds one or more members to a set.
func (s *Store) Sadd(key string, members []string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	var set map[string]struct{}
	if ok {
		if item.Type != TypeSet {
			delete(s.items, key)
			set = make(map[string]struct{})
		} else {
			set = item.Value.(map[string]struct{})
		}
	} else {
		set = make(map[string]struct{})
	}
	addedCount := 0
	for _, member := range members {
		if _, exists := set[member]; !exists {
			set[member] = struct{}{}
			addedCount++
		}
	}
	s.items[key] = Item{Value: set, Type: TypeSet, Expiration: item.Expiration}
	return addedCount
}

// Srem removes one or more members from a set.
func (s *Store) Srem(key string, members []string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeSet || s.isExpired(item) {
		return 0
	}

	set := item.Value.(map[string]struct{})
	removedCount := 0
	for _, member := range members {
		if _, exists := set[member]; exists {
			delete(set, member)
			removedCount++
		}
	}
	if len(set) == 0 {
		delete(s.items, key)
	} else {
		s.items[key] = Item{Value: set, Type: TypeSet, Expiration: item.Expiration}
	}
	return removedCount
}

// Smembers returns all members of the set.
func (s *Store) Smembers(key string) []string {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok || item.Type != TypeSet || s.isExpired(item) {
		return nil
	}

	set := item.Value.(map[string]struct{})
	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	return members
}

// Sismember checks if a member exists in a set.
func (s *Store) Sismember(key string, member string) bool {
	lock := s.getLock(key)
	lock.RLock()
	item, ok := s.items[key]
	lock.RUnlock()

	if !ok || item.Type != TypeSet || s.isExpired(item) {
		return false
	}

	set := item.Value.(map[string]struct{})
	_, exists := set[member]
	return exists
}

// HSet sets a value for a field in a hash stored at key.
func (s *Store) HSet(key string, field string, value string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	var hash map[string]string
	if ok {
		if item.Type != TypeHash {
			// If key exists but is not a hash, delete it and start a new hash.
			delete(s.items, key)
			hash = make(map[string]string)
		} else {
			// Key exists and is a hash, so get it.
			hash = item.Value.(map[string]string)
		}
	} else {
		// Key doesn't exist, create a new hash.
		hash = make(map[string]string)
	}

	// Check if the field already exists to return the correct count.
	addedCount := 0
	if _, exists := hash[field]; !exists {
		addedCount = 1
	}

	hash[field] = value
	s.items[key] = Item{Value: hash, Type: TypeHash, Expiration: item.Expiration}
	return addedCount
}

// HGet retrieves the value associated with field in the hash stored at key.
func (s *Store) HGet(key string, field string) (string, bool) {
	lock := s.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeHash || s.isExpired(item) {
		return "", false
	}

	hash := item.Value.(map[string]string)
	value, exists := hash[field]
	return value, exists
}

// HDel deletes one or more fields from the hash stored at key.
func (s *Store) HDel(key string, fields []string) int {
	lock := s.getLock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeHash || s.isExpired(item) {
		return 0
	}

	hash := item.Value.(map[string]string)
	deletedCount := 0
	for _, field := range fields {
		if _, exists := hash[field]; exists {
			delete(hash, field)
			deletedCount++
		}
	}

	// If the hash becomes empty, delete the key itself.
	if len(hash) == 0 {
		delete(s.items, key)
	} else {
		s.items[key] = Item{Value: hash, Type: TypeHash, Expiration: item.Expiration}
	}

	return deletedCount
}

// HGetAll retrieves all fields and values of the hash stored at key.
func (s *Store) HGetAll(key string) map[string]string {
	lock := s.getLock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if !ok || item.Type != TypeHash || s.isExpired(item) {
		return nil
	}

	hash := item.Value.(map[string]string)
	// Return a copy to prevent external modifications.
	newHash := make(map[string]string, len(hash))
	for k, v := range hash {
		newHash[k] = v
	}
	return newHash
}

// activeExpirationWorker performs active expiration in the background.
// It wakes up periodically to sample and delete expired keys.
func (s *Store) activeExpirationWorker() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for range ticker.C {
		keysToDelete := []string{}

		// To safely iterate over the map while other goroutines are writing,
		// we must acquire a lock for each shard before iterating over the items in that shard.
		// Since the user's current implementation uses a single map, a full lock is needed for iteration.
		// However, a range over the map is a problem. The most correct way to fix this with
		// the user's code is to add a global lock to protect the entire map during iteration.
		// The `Del` method will handle its own locking.

		// Acquire write locks for all shards to ensure no concurrent writes occur during iteration.
		for i := range s.locks {
			s.locks[i].Lock()
		}

		// Now it's safe to iterate the entire map.
		for key, item := range s.items {
			if s.isExpired(item) {
				keysToDelete = append(keysToDelete, key)
			}
		}

		// Release all the locks.
		for i := range s.locks {
			s.locks[i].Unlock()
		}

		// Delete the expired keys. The `s.Del(key)` call inside this loop
		// will acquire the specific key's lock, ensuring safety.
		deletedCount := 0
		for _, key := range keysToDelete {
			if s.Del(key) {
				deletedCount++
			}
		}

		if deletedCount > 0 {
			log.Printf("Active expiration worker: deleted %d expired keys.", deletedCount)
		}
	}
}
