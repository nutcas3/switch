package network

import (
	"fmt"
	"net"
	"sync"
)

// ConnectionPool manages active connections
type ConnectionPool struct {
	connections map[string]*net.Conn
	mutex       sync.RWMutex
	maxSize     int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*net.Conn),
		maxSize:     maxSize,
	}
}

// Add adds a connection to the pool
func (cp *ConnectionPool) Add(id string, conn *net.Conn) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if len(cp.connections) >= cp.maxSize {
		return fmt.Errorf("connection pool is full")
	}

	cp.connections[id] = conn
	return nil
}

// Remove removes a connection from the pool
func (cp *ConnectionPool) Remove(id string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	delete(cp.connections, id)
}

// Size returns the current pool size
func (cp *ConnectionPool) Size() int {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return len(cp.connections)
}
