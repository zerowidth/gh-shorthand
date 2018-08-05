package rpc

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
)

// how long to wait before giving up on the backend
const socketTimeout = 100 * time.Millisecond

// Query executes a query against the RPC server.
//
// Returns a Result if the RPC call completed successfully, regardless of
// whether the ultimate value is ready or not.
func Query(cfg config.Config, endpoint, query string) (Result, error) {
	var res Result

	if len(cfg.SocketPath) == 0 {
		return Result{Complete: true}, nil // RPC isn't enabled, don't worry about it
	}

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "unix", cfg.SocketPath)
			},
		},
		Timeout: socketTimeout,
	}

	u, err := url.Parse("http://gh-shorthand" + endpoint)
	if err != nil {
		return res, err
	}
	v := url.Values{}
	v.Set("q", query)
	u.RawQuery = v.Encode()

	resp, err := c.Get(u.String())
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}
