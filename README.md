# gitsync

`gitsync` helps manage multiple git repositories which share the same files
(or parts of them) and need them kept up-to-date and in sync.

This project was born out of the need to manage multiple repositories which
shared the same development files, like linter configs, CI/CD scripts, etc.

Why not use submodules?

- They won't do the job if there are slight differences between root and
  synchronized files.
- Some tools depend on their config files being in specific places,
  like in the root of the repository.
  While this can be solved by linking, the solution is not platform-agnostic.

That being said, I wanted a tool which would automate the process of manually
synchronizing changes to these files between repositories.

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

## Usage

`gitsync` ships with two commands:

1. `sync` - interactively creates a patch and applies it to the synchronized
repositories' files.
2. `diff` - shows the differences between the root and synchronized files in
unified format.

```shell
gitsync -c config.json [diff|sync]
```

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

`diff` runs the same as `sync`, but instead of applying the patch it simply prints it.

### Config file

The config file is a JSON file which describes the synchronization process.

```json5
{
  // Optional. Default: $HOME/.config/gitsync or $XDG_CONFIG_HOME/gitsync.
  "storePath": "/home/mh/.config/gitsync",
  // Required.
  "root": {
    // Required.
    "name": "template",
    // Required.
    "url": "https://github.com/nobl9/go-repo-template"
  },
  // Optional.
  "ignore": [
    {
      // Optional.
      "regex": "^\\s*local-prefixes:"
    }
  ],
  // Required. At least one repository must be provided.
  "syncRepositories": [
    {
      // Required.
      "name": "nobl9-go",
      // Required.
      "url": "https://github.com/nobl9/nobl9-go",
      // Optional.
      "ref": "main",
      // Optional, merged with global 'ignore' section.
      "ignore": [
        {
          // Optional.
          "regex": "custom-regex",
          // Optional.
          "hunk": ""
        }
      ],
    },
    {
      "name": "sloctl",
      "url": "https://github.com/nobl9/sloctl",
    }
  ],
  // Required. At least one file must be provided.
  "syncFiles": [
    {
      // Required.
      "name": "golangci linter config",
      // Required.
      "path": ".golangci.yml"
    }
  ]
}
```
