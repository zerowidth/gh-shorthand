package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/zerowidth/gh-shorthand/alfred"
	"github.com/zerowidth/gh-shorthand/config"
)

func init() {
	RootCmd.AddCommand(projectCommand)
}

var projectCommand = &cobra.Command{
	Use:   "project <action>",
	Short: "Project mode",
	Long: `Generates a list of Alfred items with the given <action>, for every
directory in the configured project paths ("project_dirs" key in the config
file). This is intended to be used as a single-shot Alfred script filter,
allowing Alfred to filter the results.

Action can be one of:

  edit   - for opening an editor with the project directory
  finder - for opening a project directory in Finder
  term   - for opening a terminal in a project directory`,

	Run: func(cmd *cobra.Command, args []string) {
		var items = []*alfred.Item{}
		var action = ""

		if len(args) > 0 {
			action = args[0]
		}

		path, _ := homedir.Expand("~/.gh-shorthand.yml")
		cfg, err := config.LoadFromFile(path)
		if err != nil {
			items = []*alfred.Item{errorItem("when loading ~/.gh-shorthand.yml", err.Error())}
		} else {
			items = projectItems(cfg, action)
		}

		printItems(items)
	},
}

func projectItems(cfg *config.Config, action string) (items []*alfred.Item) {
	switch action {
	case "edit":
		items = append(items, actionItems(cfg.ProjectDirMap(), "ghe", "edit", "Edit", editorIcon)...)
	case "finder":
		items = append(items, actionItems(cfg.ProjectDirMap(), "gho", "finder", "Open Finder in", editorIcon)...)
	case "term":
		items = append(items, actionItems(cfg.ProjectDirMap(), "ght", "term", "Open terminal in", editorIcon)...)
	}
	return
}

func actionItems(dirs map[string]string, uidPrefix, action, desc string, icon *alfred.Icon) (items []*alfred.Item) {
	for short, expanded := range dirs {
		for _, dirname := range findProjectDirs(expanded) {
			items = append(items, &alfred.Item{
				UID:   uidPrefix + ":" + filepath.Join(short, dirname),
				Title: desc + " " + filepath.Join(short, dirname),
				Arg:   action + " " + filepath.Join(expanded, dirname),
				Valid: true,
				Icon:  icon,
			})
		}
	}
	return
}

func findProjectDirs(root string) (dirs []string) {
	fmt.Fprintf(os.Stderr, "reading entries in %s\n", root)
	if entries, err := ioutil.ReadDir(root); err == nil {
		for _, entry := range entries {
			fmt.Fprintf(os.Stderr, "  checking %s\n", entry.Name())
			if entry.IsDir() {
				dirs = append(dirs, entry.Name())
			}
		}
	}
	return
}
