package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
)

// RPC is a set of RPC http handlers
type RPC struct {
	cfg *config.Config
}

// NewRPC creates a new RPC server with the given config
func NewRPC(cfg *config.Config) *RPC {
	return &RPC{
		cfg: cfg,
	}
}

// Mount routes the RPC handlers on a mux
func (rpc *RPC) Mount(mux *chi.Mux) {
	mux.Get("/", rpc.testHandler)
}

func (rpc *RPC) testHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}
	fmt.Fprintf(w, "query: %#v\n", r.Form.Get("q"))
}
