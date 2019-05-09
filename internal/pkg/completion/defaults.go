package completion

import "github.com/zerowidth/gh-shorthand/pkg/alfred"

var (
	defaultItems = []alfred.Item{
		alfred.Item{
			Title:        "Open repositories and issues on GitHub",
			Autocomplete: " ",
			Icon:         repoIcon,
		},
		alfred.Item{
			Title:        "List and search issues in a GitHub repository",
			Autocomplete: "i ",
			Icon:         issueListIcon,
		},
		alfred.Item{
			Title:        "List and open projects on GitHub repositories or organizations",
			Autocomplete: "p ",
			Icon:         projectIcon,
		},
		alfred.Item{
			Title:        "Search issues across GitHub",
			Autocomplete: "s ",
			Icon:         searchIcon,
		},
		alfred.Item{
			Title:        "New issue in a GitHub repository",
			Autocomplete: "n ",
			Icon:         newIssueIcon,
		},
		alfred.Item{
			Title:        "Open a project",
			Autocomplete: "e ",
			Icon:         editorIcon,
		},
	}
)
