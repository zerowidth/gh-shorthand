package rpc

import (
	"encoding/json"
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
	cache   *cache.Cache
	github  *GitHubClient
	m       sync.Mutex
	pending map[string]struct{}
}

type rpcCall func(result *Result, query string) error

// NewHandler creates a new RPC handler with the given config
func NewHandler(cfg config.Config) *Handler {
	handler := Handler{
		cache:   cache.New(resultTTL, sweepInterval),
		pending: make(map[string]struct{}),
		github:  NewGitHubClient(cfg),
	}
	return &handler
}

// Mount routes the RPC handlers on a mux
func (h *Handler) Mount(mux *chi.Mux) {
	mux.Get("/repo", h.rpcHandler(h.github.GetRepo))
	mux.Get("/issue", h.rpcHandler(h.github.GetIssue))
	mux.Get("/issues", h.rpcHandler(h.github.GetIssues))
	mux.Get("/project", h.rpcHandler(h.github.GetProject))
}

// rpcHandler creates an http handler func to wrap a GitHub API call with
// asynchronous execution and caching of its result.
func (h *Handler) rpcHandler(rpc rpcCall) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(400), 400)
			return
		}

		query := r.Form.Get("q")
		if len(query) == 0 {
			return
		}

		// Now that basic checks are done, lock the cache and pending map to see
		// if the request is already in flight. If not, kick it off.
		h.m.Lock()
		defer h.m.Unlock()
		var res Result

		if _, pending := h.pending[query]; !pending {
			if cr, ok := h.cache.Get(query); ok {
				res = cr.(Result)
			} else {
				// this will wait on the mutex immediately, but we're returning soon anyway
				go h.makeRequest(rpc, query)
			}
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			log.Fatalf(err.Error())
		}
	}
}

func (h *Handler) makeRequest(rpc rpcCall, query string) {
	h.m.Lock()
	h.pending[query] = struct{}{}
	h.m.Unlock()

	var res Result
	ttl := resultTTL

	log.Println("RPC request: ", query)
	err := rpc(&res, query)
	if err != nil {
		res.Error = err.Error()
		ttl = errorTTL
	}
	log.Printf("RPC result: %+v\n", res)

	h.m.Lock()
	delete(h.pending, query)
	res.Complete = true
	h.cache.Set(query, res, ttl)
	h.m.Unlock()
}
