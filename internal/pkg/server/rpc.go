package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
)

// RPCHandler is a set of RPC http handlers
type RPCHandler struct {
	cfg *config.Config
}

// NewRPCHandler creates a new RPC server with the given config
func NewRPCHandler(cfg *config.Config) *RPCHandler {
	return &RPCHandler{
		cfg: cfg,
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
	fmt.Fprintf(w, "query: %#v\n", query)
}
