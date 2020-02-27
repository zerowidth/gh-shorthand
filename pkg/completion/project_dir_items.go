package completion

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/sahilm/fuzzy"
	"github.com/zerowidth/gh-shorthand/pkg/alfred"
)

type projectDirMode int

const (
	modeEdit projectDirMode = iota
	modeTerm
)

func projectDirItems(searchPaths []string, search string, mode projectDirMode) (items alfred.Items) {
	// shortened path names of projects found
	projects := []string{}
	// map to the full expanded/absolute path for projects
	projectPaths := map[string]string{}

	home, _ := homedir.Dir()
	for _, searchPath := range searchPaths {
		root, err := homedir.Expand(searchPath)

		if err != nil {
			items = append(items, ErrorItem("Invalid project directory: "+searchPath, err.Error()))
			continue
		}

		projectDirs, err := findProjectDirs(root)
		if err != nil {
			items = append(items, ErrorItem("Invalid project directory: "+searchPath, err.Error()))
			continue
		}

		for _, pd := range projectDirs {
			absolute, err := filepath.Abs(pd)
			if err != nil {
				continue
			}
			// if homedir was expanded, re-shorten the path
			if root != searchPath {
				short := strings.Replace(pd, home, "~", 1)
				projectPaths[short] = absolute
				projects = append(projects, short)
			} else {
				projectPaths[pd] = absolute
				projects = append(projects, pd)
			}
		}
	}

	// filter projects by fuzzy search, if applicable
	if len(search) > 0 {
		filtered := fuzzy.Find(search, projects)
		projects = []string{}
		for _, result := range filtered {
			projects = append(projects, result.Str)
		}
	}

	for _, short := range projects {
		var item = alfred.Item{
			Title: short,
			Valid: true,
			Text:  &alfred.Text{Copy: projectPaths[short], LargeType: projectPaths[short]},
			Mods: &alfred.Mods{
				Alt: &alfred.ModItem{
					Valid:     true,
					Arg:       projectPaths[short],
					Subtitle:  "Open finder in " + short,
					Icon:      finderIcon,
					Variables: alfred.Variables{"action": "finder"},
				},
			},
		}

		if mode == modeEdit {
			item.UID = "ghe:" + short
			item.Subtitle = "Edit " + short
			item.Arg = projectPaths[short]
			item.Variables = alfred.Variables{"action": "edit"}
			item.Icon = editorIcon
			item.Mods.Cmd = &alfred.ModItem{
				Valid:     true,
				Arg:       projectPaths[short],
				Subtitle:  "Open terminal in " + short,
				Icon:      terminalIcon,
				Variables: alfred.Variables{"action": "term"},
			}
		} else {
			item.UID = "ght:" + short
			item.Subtitle = "Open terminal in " + short
			item.Arg = projectPaths[short]
			item.Variables = alfred.Variables{"action": "term"}
			item.Icon = terminalIcon
			item.Mods.Cmd = &alfred.ModItem{
				Valid:     true,
				Arg:       projectPaths[short],
				Subtitle:  "Edit " + short,
				Icon:      editorIcon,
				Variables: alfred.Variables{"action": "edit"},
			}
		}

		items = append(items, item)
	}

	return
}

func findProjectDirs(root string) ([]string, error) {
	entries, err := filepath.Glob(root + "/*")
	if err != nil {
		return []string{}, err
	}

	// if the glob didn't find anything, make sure it's targeting a valid existing
	// directory:
	if len(entries) == 0 {
		if _, err = ioutil.ReadDir(root); err != nil {
			return []string{}, err
		}
	}

	// filter out things that aren't directories
	dirs := []string{}
	for _, entryPath := range entries {
		entry, err := os.Stat(entryPath)
		if err != nil {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entryPath)
		} else if entry.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(entryPath)
			if err != nil {
				continue
			}

			if !path.IsAbs(link) {
				if link, err = filepath.Abs(path.Join(entryPath, link)); err != nil {
					continue
				}
			}

			linkInfo, err := os.Stat(link)
			if err != nil {
				continue
			}
			if linkInfo.IsDir() {
				dirs = append(dirs, entryPath)
			}
		}
	}

	return dirs, nil
}
