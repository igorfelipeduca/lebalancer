package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend interface {
	SetAlive(bool)
	IsAlive() bool
	GetURL() *url.URL
	GetActiveConnections() int
	Serve(http.ResponseWriter, *http.Request)
}

type backend struct {
	url          *url.URL
	alive        bool
	mux          *sync.RWMutex
	connections  int
	reverseProxy *httputil.ReverseProxy
}

type ServerPool interface {
	GetBackends() []Backend
	GetNextValidPeer() Backend
	AddBackend(Backend)
	GetServerPoolSize() int
}

// the round-robing pool is a load balancing technique
// this round system distributes all incoming request in different servers in a circular way
type roundRobinServerPool struct {
	backends []Backend
	mux      sync.RWMutex // we are using mutex to ensure there wouldn't be any race conditions while readin/changing the robin server pool
	current  int
	pool     ServerPool
}

func (s *roundRobinServerPool) Rotate() Backend {
	s.mux.Lock()                                             // locking the mutex so this will be the only one thread reading the server pool
	s.current = (s.current + 1) % s.pool.GetServerPoolSize() // we're rotating the request to the next available server pool
	s.mux.Unlock()                                           // unlocking the mutex after the reading

	return s.backends[s.current] // returns the next available server pool
}

func (s *roundRobinServerPool) GetNextValidPeer() Backend {
	for i := 0; i < s.pool.GetServerPoolSize(); i++ {
		nextPeer := s.Rotate()
		if nextPeer.IsAlive() {
			return nextPeer
		}
	}
	return nil
}
