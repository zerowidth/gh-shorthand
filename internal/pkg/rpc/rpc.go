package rpc

// Result is the result of an RPC call
type Result struct {
	Complete bool   `json:"complete"` // is the request finished?
	Value    string `json:"value"`    // the value, if successful
	Error    string `json:"error"`    // server error, if applicable
}
