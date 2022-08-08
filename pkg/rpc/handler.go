package rpc

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/kardianos/service"
	"github.com/patrickmn/go-cache"
	"github.com/zerowidth/gh-shorthand/pkg/config"
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
	logger  service.Logger
	m       sync.Mutex
	pending map[string]struct{}
}

type rpcCall func(result *Result, query string) error

// NewHandler creates a new RPC handler with the given config
func NewHandler(cfg config.Config, lg service.Logger) *Handler {
	handler := Handler{
		cache:   cache.New(resultTTL, sweepInterval),
		pending: make(map[string]struct{}),
		github:  NewGitHubClient(cfg),
		logger:  lg,
	}
	return &handler
}

// Mount routes the RPC handlers on a mux
func (h *Handler) Mount(mux *chi.Mux) {
	mux.Get("/repo", h.rpcHandler("repo", h.github.GetRepo))
	mux.Get("/issue", h.rpcHandler("issue", h.github.GetIssue))
	mux.Get("/issues", h.rpcHandler("issues", h.github.GetIssues))
	mux.Get("/project", h.rpcHandler("project", h.github.GetProject))
	mux.Get("/projects", h.rpcHandler("projects", h.github.GetProjects))
}

// rpcHandler creates an http handler func to wrap a GitHub API call with
// asynchronous execution and caching of its result.
func (h *Handler) rpcHandler(action string, rpc rpcCall) http.HandlerFunc {
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

		key := action + ":" + query
		if _, pending := h.pending[key]; !pending {
			if cr, ok := h.cache.Get(key); ok {
				res = cr.(Result)
			} else {
				// this will wait on the mutex immediately, but we're returning soon anyway
				go h.makeRequest(rpc, query, key)
			}
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			_ = h.logger.Error("encoding error", err)
		}
	}
}

func (h *Handler) makeRequest(rpc rpcCall, query, key string) {
	h.m.Lock()
	h.pending[key] = struct{}{}
	h.m.Unlock()

	var res Result
	ttl := resultTTL

	_ = h.logger.Infof("RPC request: %s", key)
	err := rpc(&res, query)
	if err != nil {
		res.Error = err.Error()
		ttl = errorTTL
	}
	res.Complete = true
	_ = h.logger.Infof("RPC result: %s %+v\n", key, res)

	h.m.Lock()
	delete(h.pending, key)
	h.cache.Set(key, res, ttl)
	h.m.Unlock()
}
