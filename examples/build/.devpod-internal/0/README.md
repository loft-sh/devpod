
# GitHub CLI (github-cli)

Installs the GitHub CLI. Auto-detects latest version and installs needed dependencies.

## Example Usage

```json
"features": {
    "ghcr.io/devcontainers/features/github-cli:1": {}
}
```

## Options

| Options Id | Description | Type | Default Value |
|-----|-----|-----|-----|
| version | Select version of the GitHub CLI, if not latest. | string | latest |
| installDirectlyFromGitHubRelease | - | boolean | true |



## OS Support

This Feature should work on recent versions of Debian/Ubuntu-based distributions with the `apt` package manager installed.

`bash` is required to execute the `install.sh` script.


---

_Note: This file was auto-generated from the [devcontainer-feature.json](https://github.com/devcontainers/features/blob/main/src/github-cli/devcontainer-feature.json).  Add additional notes to a `NOTES.md`._
