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

// Client represents an RPC client
type Client struct {
	socketPath string
}

// NewClient creates a new Client from a config
func NewClient(cfg config.Config) Client {
	return Client{
		socketPath: cfg.SocketPath,
	}
}

// how long to wait before giving up on the backend
const socketTimeout = 100 * time.Millisecond

// Query executes a query against the RPC server.
//
// Returns a Result if the RPC call completed successfully, regardless of
// whether the ultimate value is ready or not.
func (c Client) Query(endpoint, query string) Result {
	var res Result

	if len(c.socketPath) == 0 {
		return Result{Complete: true} // RPC isn't enabled, don't worry about it
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "unix", c.socketPath)
			},
		},
		Timeout: socketTimeout,
	}

	u, err := url.Parse("http://gh-shorthand" + endpoint)
	if err != nil {
		res.Complete = true
		res.Error = "url parsing error: " + err.Error()
		return res
	}
	v := url.Values{}
	v.Set("q", query)
	u.RawQuery = v.Encode()

	resp, err := httpClient.Get(u.String())
	if err != nil {
		res.Error = "RPC service error: " + err.Error()
		res.Complete = true
		return res
	}
	if resp.StatusCode >= 400 {
		res.Error = "RPC service error: " + resp.Status
		res.Complete = true
		return res
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		res.Error = "RPC response error: " + err.Error()
		res.Complete = true
		return res
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		res.Error = "unmarshal error: " + err.Error()
		res.Complete = true
		return res
	}

	return res
}
