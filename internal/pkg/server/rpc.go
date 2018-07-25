package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/patrickmn/go-cache"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
)

const (
	resultTTL     = 10 * time.Minute
	errorTTL      = 10 * time.Second
	sweepInterval = 10 * time.Minute
)

// RPCHandler is a set of RPC http handlers
type RPCHandler struct {
	cfg   *config.Config
	cache *cache.Cache
}

// NewRPCHandler creates a new RPC server with the given config
func NewRPCHandler(cfg *config.Config) *RPCHandler {
	return &RPCHandler{
		cfg:   cfg,
		cache: cache.New(resultTTL, sweepInterval),
	}
}

// Mount routes the RPC handlers on a mux
func (rpc *RPCHandler) Mount(mux *chi.Mux) {
	mux.Get("/", rpc.testHandler)
}

func (rpc *RPCHandler) testHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	query := r.Form.Get("q")
	if len(query) == 0 {
		return
	}

	var req Request
	cr, ok := rpc.cache.Get(query)
	if ok {
		req = *cr.(*Request)
	} else {
		req = Request{}
		go rpc.makeRequest(&req, query)
		rpc.cache.Set(query, &req, resultTTL)
	}

	if req.Done() {
		if err := req.Error(); err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, "request error: %s", err.Error())
			return
		}

		fmt.Fprintf(w, "request value: %s", req.Value())
		return
	}

	w.WriteHeader(204) // No Content (request is pending)
}

func (rpc *RPCHandler) makeRequest(req *Request, query string) {
	log.Println("RPC request: ", query)
	<-time.After(3 * time.Second)
	if query == "error" {
		log.Println("RPC request error: ", query)
		req.setError(fmt.Errorf("an error occurred in the rpc service"))
		return
	}
	log.Println("RPC result: ", query)
	req.setValue(fmt.Sprintf("RPC sees: %s", query))
}

// Request represents a pending or completed RPC request
type Request struct {
	done  bool
	value string
	error error
	m     sync.RWMutex
}

// Done checks if this RPC request is done yet
func (r *Request) Done() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.done
}

// Value retrieves the request's value, if applicable
func (r *Request) Value() string {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.value
}

// Error retrieves the request's error, if applicable
func (r *Request) Error() error {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.error
}

func (r *Request) setValue(v string) {
	r.m.Lock()
	r.value = v
	r.done = true
	r.m.Unlock()
}

func (r *Request) setError(e error) {
	r.m.Lock()
	r.error = e
	r.done = true
	r.m.Unlock()
}
