# GravityCLI

GravityCLI is an interactive terminal client for common Git and GitHub workflows.
It gives you a dashboard for authentication, repository cloning and management,
branch switching, pull request work, and a local directory/git cockpit.

## Features

- Interactive dashboard built with Bubble Tea and Lip Gloss
- GitHub authentication with OAuth device flow or a personal access token
- Search and clone authenticated GitHub repositories
- Create, edit, and delete GitHub repositories
- Switch or create local Git branches
- View, open, create, and checkout pull requests
- Navigate folders, stage files, commit, and push from a TUI

## Installation

### From source

```bash
git clone https://github.com/Joeboi01/GravityCLI.git
cd GravityCLI
go build -o gravity .
```

Then run:

```bash
./gravity
```

On Windows PowerShell:

```powershell
.\gravity.exe
```

### Releases

Prebuilt binaries should be published through GitHub Releases. Build artifacts
are intentionally not committed to the repository.

## Usage

Run the app without arguments to open the main dashboard:

```bash
gravity
```

You can also jump directly into a workflow:

```bash
gravity auth
gravity clone
gravity repo
gravity branches
gravity pr
gravity nav
```

## Authentication

GravityCLI supports two authentication modes:

- OAuth device flow: paste a GitHub OAuth client ID, then authorize in the browser.
- Personal access token: paste a token with the scopes needed for the workflows you use.

Credentials are stored in your OS user config directory with restrictive file
permissions. A future version should use the platform keychain for stronger
secret storage.

## Development

```bash
go test ./...
go vet ./...
go run .
```

## Roadmap

- Store credentials in the OS keychain
- Publish signed release binaries with checksums
- Add GitHub Actions workflow status views
- Add issue management
- Add broader test coverage for TUI flows

## License

MIT
