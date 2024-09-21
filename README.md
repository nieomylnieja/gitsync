# gitsync

`gitsync` helps manage multiple git repositories which share the same files
(or parts of them) and helps keep them up-to-date and in sync.

This project was born out of the need to manage multiple repositories which
shared the same development files, like linter configs, CI/CD scripts, etc.

Why not use submodules?

- They won't be sufficient if there are slight differences between root and
  synchronized files and If the tool does not support merging multiple
  configs.
- Some tools depend on their config files being in specific places,
  like in the root of the repository.
  While this can be solved by linking, the solution is not platform-agnostic.

What I ended up doing was:

- Define a _root_ repository, which would serve as a staple for other
  repositories to follow.
- Update linter configs, CI/CD scripts, GH workflows, etc. at the _root_
  first and only then propagate these changes manually to all the other
  _synchronized_ repositories.

However, with a growing number of repositories I had to govern, it started
to become more and more painful and time consuming to do that by hand.

Enter `gitsync` 😉

## Install

Use pre-built binaries from
the [latest release](https://github.com/nieomylnieja/gitsync/releases/latest)
or install
with Go:

```shell
go install github.com/nieomylnieja/gitsync/cmd/gitsync@latest
```

It can also be built directly from this repository:

```shell
git clone https://github.com/nieomylnieja/gitsync.git
cd gitsync
make build
./bin/gitsync ./go.mod
```

### Requirements

The following programs must be installed and available in the `$PATH`:

- `git`
- `diff` (GNU version)
- `gh` (GitHub CLI)

## Usage

`gitsync` ships with two commands:

1. `sync` - interactively creates a patch and applies it to the synchronized
   repositories' files.
2. `diff` - shows the differences between the root and synchronized files in
   unified format.

```shell
gitsync -c config.json [diff|sync]
```

If the `-c` (config file path) flag is not provided,
`gitsync` will look for a `gitsync.json` file in either
`$XDG_CONFIG_HOME/gitsync/config.json` or
`$HOME/.config/gitsync/config.json`.

### Sync

`sync` performs the following steps:

1. Clones the repository (if not yet cloned) into `storePath`
   (see [config](#config-file)).
2. Fetches the latest changes from the root repository.
3. Interactively creates a patch between the root and synchronized files.
    - The user can choose to skip hunks.
    - The user can choose to permanently ignore hunks, by:
        - Adding `regex` rules to the `ignore` field in the config file.
        - Choosing `i` option in the prompt, which will add `hunk` rules to the
          `ignore` field in the config file.
        - Manually adding `hunk` rules to the `ignore` field in the config file.
4. Applies the patch to the synchronized repository.
5. Commits the changes to the index.
6. Pushes the changes to the remote repository.
7. Creates a pull request (currently only GitHub is supported).

### Diff

`diff` runs the same as `sync`, but instead of applying the patch it simply
prints it.

### Config file

The config file is a JSON file which describes the synchronization process.

```json5
{
  // Optional. Default: $HOME/.local/share/gitsync or $XDG_DATA_HOME/gitsync.
  // Path to the directory where the repositories will be cloned and stored.
  "storePath": "~/.config/gitsync",
  // Required. Configuration of the root repository.
  // Follows the same format as syncRepositories[].
  "root": {
    "name": "template",
    "url": "https://github.com/nieomylnieja/go-repo-template.git"
  },
  // Optional.
  "ignore": [
    // If neither 'repositoryName' nor 'fileName' is provided,
    // the rule will apply globally.
    // If both are provided, the rule will apply only to the specific repository file.
    {
      // Optional. Name of the repository to which the ignore rule applies.
      "repositoryName": "go-libyear",
      // Optional. Name of the file to which the ignore rule applies.
      "fileName": "golangci linter config",
      // Optional. List of regular expressions used to ignore matching hunks.
      // Note: This regular expression is passed to 'diff -I <regex>' and thus follows
      // BRE (basic regular expression) rules, you may need to escape some characters, like '+'.
      // Ref: https://www.gnu.org/software/grep/manual/html_node/Basic-vs-Extended.html.
      "regex": ["^\\s\\+local-prefixes:"]
    },
    {
      // Optional. Hunks to be ignored are represented with lines header and changes list.
      // Either enter it manually or use the 'i' option in the sync command prompt.
      //
      // Ref: https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html.
      "hunks": [{
        // Optional. If lines are not provided the changes will be matched anywhere within the file.
        "lines": "@@ -3,0 +4,2 @@",
        // Required.
        "changes": [
          "+  skip-dirs:",
          "+    - scripts"
        ]
      }]
    }
  ],
  // Required. At least one repository must be provided.
  "syncRepositories": [
    {
      // Required. Name of the repository, must be unique.
      "name": "go-libyear",
      // Required. URL used to clone the repository.
      "url": "https://github.com/nieomylnieja/go-libyear.git",
      // Optional. Default: "origin/main".
      "ref": "dev-branch"
    },
    {
      "name": "sword-to-obsidian",
      "url": "https://github.com/nieomylnieja/sword-to-obsidian.git"
    }
  ],
  // Required. At least one file must be provided.
  "syncFiles": [
    {
      // Required. Descriptive name of the file.
      "name": "golangci linter config",
      // Required. Relative path to the file in both root and synchronized repositories.
      "path": ".golangci.yml"
    }
  ]
}
```
