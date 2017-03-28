package alfred

// Items is a list of items to return to Alfred
type Items struct {
	Items []Item  `json:"items"`
	Rerun float32 `json:"rerun,omitempty"`
}

// ByTitle provides a sorting interface for stable output
type ByTitle []Item

func (a ByTitle) Len() int           { return len(a) }
func (a ByTitle) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTitle) Less(i, j int) bool { return a[i].Title < a[j].Title }

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

// Icon is a custom icon for an item
type Icon struct {
	Path string `json:"path"`           // the path to a file
	Type string `json:"type,omitempty"` // optional, "fileicon" for a path, "filetype" for a specific file
}
