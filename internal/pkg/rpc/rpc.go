package rpc

// Result is the result of an RPC call
type Result struct {
	Complete bool   `json:"complete"` // is the request finished?
	Error    string `json:"error"`    // server error, if applicable

	Repos  []Repo  `json:"repos"`
	Issues []Issue `json:"issues"`
}

// Repo is a respository in an RPC result
type Repo struct {
	Description string `json:"description"`
}

// Issue is an issue in a RPC result
type Issue struct {
	Type  string `json:"type"`
	Title string `json:"description"`
	State string `json:"state"`
}
