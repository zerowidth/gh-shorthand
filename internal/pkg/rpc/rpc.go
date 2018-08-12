package rpc

// Result is the result of an RPC call
type Result struct {
	Complete bool   `json:"complete"` // is the request finished?
	Error    string `json:"error"`    // server error, if applicable

	Repos []Repo `json:"repos"`
}

// Repo is a respository in an RPC result
type Repo struct {
	Description string `json:"description"`
}
