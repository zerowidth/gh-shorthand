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
	cfg     *config.Config
	cache   *cache.Cache
	m       sync.Mutex
	pending map[string]struct{}
}

// NewRPCHandler creates a new RPC server with the given config
func NewRPCHandler(cfg *config.Config) *RPCHandler {
	handler := RPCHandler{
		cfg:     cfg,
		cache:   cache.New(resultTTL, sweepInterval),
		pending: make(map[string]struct{}),
	}
	return &handler
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

	// now that basic checks are done, lock the cache and pending map to see
	// if the request is already in flight.
	rpc.m.Lock()
	defer rpc.m.Unlock()
	var res Result

	if _, ok := rpc.pending[query]; ok {
		w.WriteHeader(204) // No Content (request is pending)
		return
	} else if cr, ok := rpc.cache.Get(query); ok {
		res = cr.(Result)
		if res.Error != nil {
			w.WriteHeader(400) // bad request
			fmt.Fprintf(w, "request error: %s", res.Error.Error())
			return
		}
		fmt.Fprintf(w, "request value: %s", res.Value)
		return
	}

	// this will lock on the mutex immediately, but we're returning soon
	go rpc.makeRequest(query)

	w.WriteHeader(204) // No Content (request is pending)
}

func (rpc *RPCHandler) makeRequest(query string) {
	rpc.m.Lock()
	rpc.pending[query] = struct{}{}
	rpc.m.Unlock()

	log.Println("RPC request: ", query)
	<-time.After(2 * time.Second)
	res := Result{Query: query}
	var ttl time.Duration
	if query == "error" {
		log.Println("RPC request error: ", query)
		res.Error = fmt.Errorf("an error occurred in the rpc service")
		ttl = errorTTL
	} else {
		log.Println("RPC result: ", query)
		res.Value = fmt.Sprintf("RPC sees: %s", query)
		ttl = resultTTL
	}

	rpc.m.Lock()
	delete(rpc.pending, query)
	rpc.cache.Set(query, res, ttl)
	rpc.m.Unlock()
}

// Result is the result of an RPC call
type Result struct {
	Query string
	Value string
	Error error
}
