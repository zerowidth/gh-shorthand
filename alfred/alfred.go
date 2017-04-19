package alfred

import "fmt"

// FilterResult is the final result of an Alfred script filter,
// to be rendered as JSON.
type FilterResult struct {
	Items     Items      `json:"items"`
	Rerun     float64    `json:"rerun,omitempty"`
	Variables *Variables `json:"variables,omitempty"`
}

// NewFilterResult provides an initialized FilterResult that contains the
// required (but empty) Items list
func NewFilterResult() *FilterResult {
	return &FilterResult{Items: Items{}}
}

// Items is a list of Item pointers.
type Items []*Item

// Variables is a map of string to string variables for alfred to pass on to a
// subsquent iteration of a script filter
type Variables map[string]string

// Item is an Alfred result item
type Item struct {
	UID          string `json:"uid,omitempty"`          // optional unique identifier for alfred to learn from
	Title        string `json:"title"`                  // title displayed in the result row
	Subtitle     string `json:"subtitle,omitempty"`     // optional subtitle displayed in the result row
	Arg          string `json:"arg,omitempty"`          // recommended string argument to pass through to output action
	Icon         *Icon  `json:"icon,omitempty"`         // optional icon argument
	Valid        bool   `json:"valid"`                  // valid means "actionable", false means "populate autocomplete text"
	Autocomplete string `json:"autocomplete,omitempty"` // recommended string to autocomplete with tab key
	// Type string // "default", "file", "file:skipcheck" to treat the result as a file
	// Mod Modifier // optional modifier keys object
	// Text string // optional text if copied to clipboard or displayed as large text
	// Quicklook string // optional url for quicklook
}

func (i *Item) String() string {
	return fmt.Sprintf("%#v", *i)
}

// AppendItems is shorthand for adding more items to a FilterResult's Items list
func (result *FilterResult) AppendItems(items ...*Item) {
	result.Items = append(result.Items, items...)
}

// SetVariable to set a variable in the result output
func (result *FilterResult) SetVariable(name, value string) {
	if result.Variables == nil {
		result.Variables = &Variables{name: value}
	} else {
		(*result.Variables)[name] = value
	}
}

// Icon is a custom icon for an item
type Icon struct {
	Path string `json:"path"`           // the path to a file
	Type string `json:"type,omitempty"` // optional, "fileicon" for a path, "filetype" for a specific file
}
