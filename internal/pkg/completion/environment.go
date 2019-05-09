package completion

import (
	"os"
	"strconv"
	"time"
)

// Environment represents the runtime environment from Alfred's invocation of
// this binary.
type Environment struct {
	Query string
	Start time.Time
}

// LoadAlfredEnvironment extracts the runtime environment from the OS environment
//
// The result of a script filter can set environment variables along with a
// "rerun this" timer for another invocation, and this retrieves and stores that
// information.
//
// Exported publicly for use with debugging
func LoadAlfredEnvironment(input string) Environment {
	e := Environment{
		Query: input,
		Start: time.Now(),
	}

	if query, ok := os.LookupEnv("query"); ok && query == input {
		if sStr, ok := os.LookupEnv("s"); ok {
			if nsStr, ok := os.LookupEnv("ns"); ok {
				if s, err := strconv.ParseInt(sStr, 10, 64); err == nil {
					if ns, err := strconv.ParseInt(nsStr, 10, 64); err == nil {
						e.Start = time.Unix(s, ns)
					}
				}
			}
		}
	}

	return e
}

// Duration since alfred saw the first query
func (e Environment) Duration() time.Duration {
	return time.Since(e.Start)
}
