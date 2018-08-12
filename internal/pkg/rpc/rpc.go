package rpc

// Result is the result of an RPC call
type Result struct {
	Complete bool   `json:"complete"` // is the request finished?
	Error    string `json:"error"`    // server error, if applicable

	Repos    []Repo    `json:"repos"`
	Issues   []Issue   `json:"issues"`
	Projects []Project `json:"projects"`
}

// Repo is a respository in an RPC result
type Repo struct {
	Description string `json:"description"`
}

// Issue is an issue in a RPC result
type Issue struct {
	Type   string `json:"type"`
	State  string `json:"state"`
	Title  string `json:"description"`
	Repo   string `json:"repo"`
	Number string `json:"number"`
}

// Project is a project in an RPC result
type Project struct {
	Name  string `json:"name"`
	State string `json:"state"`
}
