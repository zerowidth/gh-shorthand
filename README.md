# gh-shorthand

A golang server and CLI utility to generate autocomplete items from shorthand input, used as the backend by the [`gh-shorthand.alfredworkflow`](https://github.com/zerowidth/gh-shorthand.alfredworkflow) Alfred workflow.

## Installation

By default, the Alfred workflow invokes the executable `~/go/bin/gh-shorthand`.

```sh
mkdir -p ~/go/bin
GOBIN="$TMPDIR" go get github.com/zerowidth/gh-shorthand/cmd
mv "$TMPDIR"/cmd ~/go/bin/gh-shorthand
```

Run `gh-shorthand --help` to make sure that it's working.

### RPC server installation

To install and run the RPC server component, use the `server` subcommand:

* `gh-shorthand server install` to install the RPC server as a launchd service
* `gh-shorthand server remove` to remove the launchd service
* `gh-shorthand server start` to start the RPC service
* `gh-shorthand server stop` to stop the RPC service
* `gh-shorthand server run` to run the RPC service in the foreground. This is useful when trying this out for the first time or during development.

Note that the RPC server will not run correctly until it's configured.

## Configuration

`gh-shorthand` expects a `~/.gh-shorthand.yml` file for its operation. The file must exist but everything in it is optional. The bare minimum configuration: `touch ~/.gh-shorthand.yml`.

### Reference config file

```yaml
---
# The default repository, if none is provided. This can be empty/unset.
# default_repo:

# The repository shorthand map
repos:
  # gs: "zerowidth/gh-shorthand"

# The user shorthand map
users:
  # z: "zerowidth"

# Project directory listing:
project_dirs:
  # - "~/code"
  # - "~/go/src/*/*"

# The command or script to open the editor.
editor: "code -n"

# GitHub API token (requires `read:org,repo,user` permission)
# enables live search results and annotations
api_token: yourtoken
```

### User/Repository shorthand and completion

#### Default repository

A default repository defines the repository that an unscoped search or shorthand will apply to:

```yaml
default_repo: "zerowidth/dotfiles"
```

When set, shorthand like `#123` resolves to `zerowidth/dotfiles#123`.

#### Repository shorthand map

This maps repository shorthand to full owner/name paths:

```yaml
repos:
  gs: zerowidth/gh-shorthand
```

This resolves `gs` to the `zerowidth/gh-shorthand` repository, so `gs 123` maps to the `zerowidth/gh-shorthand#123` issue.

#### User shorthand map

Similar to repo shorthand, this resolves shorthand usernames.

```yaml
users:
  z: zerowidth
```

This resolves `z/gh-shorthand` to the `zerowidth/gh-shorthand` repository.

### Project directory configuration

For the "edit project" and "open in terminal" actions, specify a list of directories.

```yaml
project_dirs:
  - "~/code"
  - "~/work/projects"
```

With the following directory tree:

```
~
├── code
│   ├── dotfiles
│   └── demo
└── work
    └── projects
        ├── client
        └── server
```

The fuzzy search string `cdf` resolves to `~/code/dotfiles`, `wc` to `~/work/projects/client`, and `w/s` to `~/work/projects/server`.

Each root directory implies a wildcard at the end: `~/code` is treated internally as `~/code/*`. You can add wildcards of your own which can be useful for `$GOPATH/src`: adding `~/go/src/github.com/*` will index both the `github.com/zerowidth/gh-shorthand` and `github.com/spf13/viper` packages in `~/go/src`. Adding another `*`, `~/go/src/*/*`, will index packages like `golang.org/x/sync` too.

### Editor configuration

Two keys are available in the config file to control how the editor is opened.

If your editor can be opened with a single command that takes a path as its argument (either to a file or a directory), the `editor` key will suffice:

```yaml
editor: "/usr/local/bin/code -n"
```

If your editor requires environment variables such as `PATH` additions or a correct current working directory when invoked, you can  specify a script that is `eval`'d in `bash` by Alfred. `$path` is set to the input path with `~` expanded to `$HOME`. An example with VSCode:

```yaml
editor_script: 'exec /usr/local/bin/zsh -l -c "/usr/local/bin/code -n $path"'
```

MacVim expects the current working directory to be set before opening a directory:

```yaml
editor_script: |
  if [ -d "$path" ]; then
    exec /usr/local/bin/zsh -l -c "cd \"$path\" && /usr/local/bin/mvim ."
  else
    exec /usr/local/bin/zsh -l -c "cd \"$(dirname "$path")\" && /usr/local/bin/mvim \"$(basename "$path")\""
  fi
```

The `editor_script` key takes precedence over `editor`.

### RPC server configuration

To enable the RPC server, set a [GitHub API token](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line):

```
api_token: yourtokenhere
```

This token requires the `read:org, repo, user` scope.

By default the `gh-shorthand` completion utility communicates with the RPC server via the unix socket at `/tmp/gh-shorthand.sock`. To override this, set the `socket_path` configuration key to a different value.

## Usage

This script is meant to be operated with the [corresponding Alfred workflow and script filter](https://github.com/zerowidth/gh-shorthand.alfredworkflow) as its frontend.

### Commands

#### `gh-shorthand complete`

The `complete` subcommand is executed by the Alfred workflow and produces [Alfred script filter JSON](https://www.alfredapp.com/help/workflows/inputs/script-filter/json/) for Alfred to render. See below for full documentation.

#### `gh-shorthand markdown-link`

This takes an input string, provided by Alfred from the contents of the clipboard, and generates a markdown link for the referenced repository or issue.

#### `gh-shorthand issue-reference`

This takes an input string, provided by Alfred from the contents of the clipboard, and generates a GitHub issue reference for a given issue URL.

#### `gh-shorthand server`

The `server` subcommand is used to manage the `gh-shorthand` RPC server.

##### `gh-shorthand server run`

Run the server in the foreground. Used for development.

##### `gh-shorthand server install`

Installs the `gh-shorthand` binary as a `launchd` service to the system. This server is used for RPC by the `complete` subcommand.

##### `gh-shorthand server remove`

Removes the `gh-shorthand` RPC service.

##### `gh-shorthand server start`

Starts the launchd service.

##### `gh-shorthand server stop`

Stops the launchd service.

##### `gh-shorthand server restart`

Restarts the launchd service

#### `gh-shorthand editor`

Emits a shell snippet for the Alfred workflow to execute which opens an editor in a `$path` set by the workflow.

## Completion

The core shorthand completion utility, the `complete` subcommand, converts input from Alfred into a list of Alfred actions for display.

The input is a single string argument consisting of one or two characters to define the mode of operation, a space, and then an optional repository definition or query. The script filter is configured to allow an optional argument, so if no argument is given to the `complete` command, it will display a list of top-level default actions.

This script outputs [Alfred script filter JSON](https://www.alfredapp.com/help/workflows/inputs/script-filter/json/) representing Alfred result items.

The Alfred JSON references icons which live in the Alfred workflow's directory. To add a new icon, refer to it it in this codebase but add the `.png` to the workflow itself. The icons were generated from an older [octicons](https://primer.style/octicons/) version, converted to PNG with the now-defunct `fa2png.io`. The hex codes used were: #6F41C0 purple, #CB2431 red, #27A745 green.

### Definitions

These are the types of input allowed:

* `repo`: a repository name, consisting of one of:
    * An explicit username and repository name: `username/repo-name`
    * Repository shorthand, if configured: `gs` from the example configuration above expands to `zerowidth/gh-shorthand`.
    * User shorthand, if configured, and a repository name: `z/repo-name` becomes `zerowidth/repo-name`
    * Repository arguments are optional when a default repository is configured.
* `user`: - a user or organization name, one of:
    * Fully qualified user or organization, e.g. `github`.
    * User shorthand: `z` from the configuration above maps to `zerowidth`.
* `issue` - an issue or pull request number, prefixed by a `#` or space if a repository argument is present: `username/repo-name 123`, `123`, or `#123`
* `project` - a project number, with the same format as `issue`
* `/path` - a relative URL path fragment, for opening specific paths under a repository: `/branches`, `/tree/master`
* `query` - freeform text, usually a search query.
* `[item]` is optional, `<item>` is required, `|` separates alternatives. All arguments are separated by spaces.

### Completion modes

The mode is defined by the first one or two characters, followed by a required space, and then the arguments for that mode.

* `(empty string)` : Display the default Alfred items.
* `(space)` : `[repo [issue|/path] | issue | /path]` : Open a repository or issue
    * Opens a repository if given or the default repository.
    * Opens an issue for a repository if given or the default repository.
    * Opens a relative path under a repository.
    * If RPC is enabled, updates the repo or issue to show its title and open/closed state.
* `i` : `[repo] [query]` : List or search issues for a repository.
    * If RPC is enabled, displays issue search results.
* `p` : `[repo | user] [project]` : List or show a project for an organization or repository. Uses the default repository if no repo or user given.
    * If RPC is enabled, displays the list of recent projects, or updates a given project to show its title and open/closed state.
* `n` : `[repo] [query]` : Create a new issue in the given repo or default repo. `query` defines the new issue's title, if provided.
* `e` : `[query]` : Edit a project directory.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `o` : `[query]` : Open a project directory in Finder.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `t` : `[query]` : Open a terminal in a project directory.
    * Fuzzy-matches the query against project directory names in the configured directories.
* `s` : `<query>` : Search all GitHub issues for the given query.
    * If RPC enabled, displays matching issues.

## RPC

If the `socket_path` is configured, `gh-shorthand complete` assumes an RPC server is available. It uses the server for retrieving search results, listing issues, or updating result items with titles, descriptions, and open/closed states.

The RPC server is a JSON over HTTP service which wraps a GraphQL client that retrieves and caches information from the [GitHub v4 API](https://developer.github.com/v4/).

Because Alfred script filters are synchronous, the Alfred input window will show no results for a script filter until they're available. In order to make the workflow as interactive as possible, it's designed to always return results as quickly as possible. This extends to the RPC handlers, which always return immediately ("ok", "not ready", or "error") instead of waiting for results from the GitHub API. To save on API call budgets, the RPC server will only issue a unique query once against the API, regardless of how many times the `gh-shorthand complete` frontend requests it. Cached results (unless they're errors) are kept in memory for a few minutes to keep lookups fast and fresh-enough.

API calls are also reduced by delaying queries until the Alfred input has paused for a short period of time, i.e. you've stopped typing. If you were to type `g 123` as the input, we don't want to make a separate API query for issues `1`, `12`, and `123`, just `123`. The delay is calculated by using environment variables sent the Alfred response along with a request to re-run the same script filter again after a short interval. Re-runs with the same input include that environment for the re-run, so `gh-shorthand` uses that to calculate elapsed time for any given input. With the input `g 123` typed into Alfred and an example query delay of 200ms, this looks something like:

* `complete ' 1'` renders undecorated results for issue 1 and starts the timer for input ` 1`. Alfred is asked to re-run the same query in 100ms (the shortest allowed interval), but a re-invocation for the same input will include the environment variables (the timer) from the first invocation.
* `complete ' 12'` renders undecorated results for issue 12, and starts the timer for input ` 12`. Alfred is asked to re-run the same query in 100ms.
* `complete ' 123'` renders undecorated results for issue 12, and starts the timer for input ` 123`. Alfred is asked to re-run in 100ms.
* `complete ' 123'` run again, now with the timer showing 100ms has elapsed. Alfred is asked to re-run in 100ms.
* `complete ' 123'` - with 200ms elapsed. Delay time has passed so it makes an RPC request. The RPC server replies "results not available", so same un-decorated results are returned with a request to re-run in 100ms. In the background, the RPC server issues a GitHub API call to fetch the relevant data.
* `complete ' 123'` - with 300ms elapsed. RPC request, reply is "results not available" so same un-decorated results are returned with re-run in 100ms. In the meantime, the API call completes.
* `complete ' 123'` - with 400ms elapsed. RPC request, reply is "data available". Decorated results are returned, with no request for Alfred to re-run the query again.

`gh-shorthand complete` also uses the environment-based timer to emit an animated text loading indicator to show that something is happening in the background. So long as the returned JSON results are stable (as in, ordering), Alfred running the same script filter over and over rapidly doesn't _appear_ that way to you. It just looks like the workflow is reacting in real-time to your input.

In short, `gh-shorthand complete` gets called frequently, but only as frequently as needed until results are available.

## Contributing

Open an issue or PR.
