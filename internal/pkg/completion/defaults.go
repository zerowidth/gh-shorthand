package completion

import "github.com/zerowidth/gh-shorthand/pkg/alfred"

var (
	repoDefaultItem = &alfred.Item{
		Title:        "Open repositories and issues on GitHub",
		Autocomplete: " ",
		Icon:         repoIcon,
	}
	issueListDefaultItem = &alfred.Item{
		Title:        "List and search issues in a GitHub repository",
		Autocomplete: "i ",
		Icon:         issueListIcon,
	}
	projectListDefaultItem = &alfred.Item{
		Title:        "List and open projects on GitHub repositories or organizations",
		Autocomplete: "p ",
		Icon:         projectIcon,
	}
	issueSearchDefaultItem = &alfred.Item{
		Title:        "Search issues across GitHub",
		Autocomplete: "s ",
		Icon:         searchIcon,
	}
	newIssueDefaultItem = &alfred.Item{
		Title:        "New issue in a GitHub repository",
		Autocomplete: "n ",
		Icon:         newIssueIcon,
	}
	editProjectDefaultItem = &alfred.Item{
		Title:        "Edit a project",
		Autocomplete: "e ",
		Icon:         editorIcon,
	}
	openFinderDefaultItem = &alfred.Item{
		Title:        "Open a project directory in Finder",
		Autocomplete: "o ",
		Icon:         finderIcon,
	}
	openTerminalDefaultItem = &alfred.Item{
		Title:        "Open terminal in a project",
		Autocomplete: "t ",
		Icon:         terminalIcon,
	}
)
