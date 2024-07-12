/*
Package selfupdate provides self-update mechanism to Go command line tools.

Go does not provide the way to install/update the stable version of tools. By default, Go command line tools are updated

- using `go get -u` (updating to HEAD)
- using system's package manager (depending on the platform)
- downloading executables from GitHub release page manually

By using this library, you will get 4th choice:

- from your command line tool directly (and automatically)

go-github-selfupdate detects the information of the latest release via GitHub Releases API and check the current version.
If newer version than itself is detected, it downloads released binary from GitHub and replaces itself.

- Automatically detects the latest version of released binary on GitHub
- Retrieve the proper binary for the OS and arch where the binary is running
- Update the binary with rollback support on failure
- Tested on Linux, macOS and Windows
- Many archive and compression formats are supported (zip, gzip, xzip, tar)

There are some naming rules. Please read following links.

Naming Rules of Released Binaries:
  https://github.com/rhysd/go-github-selfupdate#naming-rules-of-released-binaries

Naming Rules of Git Tags:
  https://github.com/rhysd/go-github-selfupdate#naming-rules-of-git-tags

This package is hosted on GitHub:
  https://github.com/rhysd/go-github-selfupdate

Small CLI tools as wrapper of this library are available also:
  https://github.com/rhysd/go-github-selfupdate/cmd/detect-latest-release
  https://github.com/rhysd/go-github-selfupdate/cmd/go-get-release
*/
package selfupdate
