package rpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	cache "github.com/patrickmn/go-cache"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
)

const (
	resultTTL     = 10 * time.Minute // how long to keep successful results
	errorTTL      = 10 * time.Second // how long to keep errors
	sweepInterval = 10 * time.Minute // how often to sweep the cache
)

// Handler is a set of RPC http handlers
type Handler struct {
	cfg     config.Config
	cache   *cache.Cache
	m       sync.Mutex
	pending map[string]struct{}
}

// NewHandler creates a new RPC handler with the given config
func NewHandler(cfg config.Config) *Handler {
	handler := Handler{
		cfg:     cfg,
		cache:   cache.New(resultTTL, sweepInterval),
		pending: make(map[string]struct{}),
	}
	return &handler
}

// Mount routes the RPC handlers on a mux
func (h *Handler) Mount(mux *chi.Mux) {
	mux.Get("/", h.testHandler)
}

func (h *Handler) testHandler(w http.ResponseWriter, r *http.Request) {
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
	h.m.Lock()
	defer h.m.Unlock()
	var res Result

	if _, ok := h.pending[query]; ok {
		_ = json.NewEncoder(w).Encode(res)
		return
	} else if cr, ok := h.cache.Get(query); ok {
		res = cr.(Result)
		if len(res.Error) > 0 {
			w.WriteHeader(500) // internal server error
		}
		_ = json.NewEncoder(w).Encode(res)
		return
	}

	// this will wait on the mutex immediately, but we're returning soon anyway
	go h.makeRequest(query)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) makeRequest(query string) {
	h.m.Lock()
	h.pending[query] = struct{}{}
	h.m.Unlock()

	log.Println("RPC request: ", query)
	<-time.After(2 * time.Second)
	res := Result{}
	var ttl time.Duration
	if query == "error" {
		log.Println("RPC request error: ", query)
		res.Error = "an error occurred in the rpc service"
		ttl = errorTTL
	} else {
		log.Println("RPC result: ", query)
		res.Value = fmt.Sprintf("RPC sees: %s", query)
		ttl = resultTTL
	}

	h.m.Lock()
	delete(h.pending, query)
	res.Complete = true
	h.cache.Set(query, res, ttl)
	h.m.Unlock()
}
