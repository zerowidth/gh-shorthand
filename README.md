# gh-shorthand

A go program to generate Alfred autocomplete items from shorthand input, used by [`gh-shorthand.alfredworkflow`](https://github.com/zerowidth/gh-shorthand.alfredworkflow).

## Installation

`go get github.com/zerowidth/gh-shorthand`, and it'll be there in `$GOPATH/bin`.

Running `gh-shorthand " zerowidth/gh-shorthand"` in a terminal (the space is important) should print out some JSON, including something about a missing configuration file.

## Configuration

`gh-shorthand` expects a `~/.gh-shorthand.yml` file for its operation. The file must exist, but everything in it is optional. The bare minimum configuration: `touch ~/.gh-shorthand.yml`.

 An example config file:

```yml
# define the default repository for gh-shorthand to use:
default_repo: "zerowidth/dotfiles"

# repository shorthand, these are abbreviations for full repositories:
repos:
  df: "zerowidth/dotfiles"
  vf: "zerowidth/vimfiles"
  gs: "zerowidth/gh-shorthand"
  gb: "zerowidth/gh-shorthand-backend"

# user shorthand
users:
  z: "zerowidth"

# local directories containing code/project directories
project_dirs:
  - "~/code"
  - "~/go/src/github.com/zerowidth"

# gh-shorthand-backend RPC configuration for live results and annotation:
socket_path: /tmp/gh-shorthand.sock
api_token: github-token-here
```

## Usage

This script is meant to be operated with the [corresponding Alfred workflow and script filter](https://github.com/zerowidth/gh-shorthand.alfredworkflow) as its frontend. 

The input is a single string argument, consisting of a single character to define the mode of operation followed by an optional repository definition or query. Because the input is for a script filter, the input to Alfred itself is presumed to start with a `gh`, which is stripped before being passed to this program. The script filter is configured to allow an optional argument, and if no argument is given, it will display the default actions (with autocomplete).

This script outputs [Alfred script filter JSON](https://www.alfredapp.com/help/workflows/inputs/script-filter/json/) representing Alfred result items. The defined actions for these results (if actionable) are in the form of `action argument`. The action is used by the workflow to determine which action to take, and the argument is the argument needed by whatever action:

* `open` - open a URL in the browser.
* `edit` - opens a file or directory in an editor.
* `finder` - opens a directory in Finder.
* `term` - opens a terminal in the given directory.
* `paste` - pastes the given argument to the foreground application, leaving it on the clipboard as well.
* `error` - display an error message (useful for debugging, mostly).

The Alfred items this script returns references icons which live in the Alfred workflow's directory. To add a new icon, reference it in this codebase, but add the .png to the workflow itself.

### Definitions

Referenced below, these are the types of input allowed:

* `repo` - a repository name, consisting of one of:
    * Repository shorthand: `df` from the configuration above, maps to `zerowidth/dotfiles`.
    * User shorthand followed by a slash and a repository name: `z/vim-bgtags` maps to `zerowidth/vim-bgtags`.
    * A fully qualified user/repository, e.g. `github/linguist`.
* `user` - a user or organization name, one of:
    * User shorthand: `z` from the configuration above maps to `zerowidth`.
    * Fully qualified user or organization, e.g. `github`.
* `issue` - an issue or pull request number. Numeric and separated from the repository name by a space.
* `project` - a project number. Numeric.
* `/path` - a relative URL path, for opening specific paths under a repository.
* `query` - freeform text, usually a search query.
* `[item]` is optional, `<item>` is required, `|` separates alternatives. All arguments are separated by spaces.

### Modes

Listed as the first character of input, which defines the mode, followed by a description of the arguments and what kind of items the script generates for Alfred:

* (empty) : Display the default items.
* `(space)` : `[repo [issue|/path] | issue | /path]` : Open a repository or issue
    * Opens a repository if given, or the default repository.
    * Opens an issue for a repository if given, or the default repository.
    * Opens a relative path under a repository if a repository is given.
    * If RPC is enabled, updates the repo or issue to show its title and open/closed state.
* `i` : `<repo> [query]` : List or search issues for a repository.
    * If RPC is enabled, displays issue search results.
* `p` : `[repo | user] [project]` : List or show a project for an organization or repository. Uses the default repository if no repo or user given.
    * If RPC is enabled, displays the list of recent projects, or updates a given project to show its title and open/closed state.
* `n` : `[repo] [query]` : Create a new issue in the given repo or default repo. `query` defines the new issue's title, if provided.
* `m` : `[repo] <issue>` : Generate a markdown link to the given issue for the given or default repository and paste it into the foreground application.
* `r` : `[repo] <issue>` : Generate an issue reference (e.g. `zerowidth/vim-bgtags#1`) and paste it into the foreground application.
* `e` : `[query]` : Edit a project directory.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `o` : `[query]` : Open a project directory in Finder.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `t` : `[query]` : Open a terminal in a project directory.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `s` : `<query>` : Search all GitHub issues for the given query.
    * If RPC enabled, displays matching issues.

## RPC

If the `socket_path` is configured, `gh-shorthand` assumes a [`gh-shorthand-backend`](https://github.com/zerowidth/gh-shorthand-backend) RPC server is available for retrieving search results or updating result items with titles and open/closed states.

Because Alfred's script filters are synchronous, they cannot block on script calls without also blocking the Alfred UI. For this reason, `gh-shorthand` is designed to return as quickly as possible. If RPC is enabled, an RPC query is sent to the RPC socket and the state (pending, complete, or error) is retrieved. If an RPC query is pending, the `gh-shorthand`'s response to Alfred includes an extra piece of metadata instructing Alfred to run the script filter again in a fraction of a second with the same query. So long as the RPC request/response is fast, the Alfred results from `gh-shorthand` will appear nearly instantly and will retry (by re-running) until the RPC query has returned a value.

## Contributing

Open an issue or PR.
