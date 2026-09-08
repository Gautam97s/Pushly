# Pushly

> Securely manage local Git repositories from your phone.

Pushly is a remote Git management system for developers who want to control repositories stored on their own computer while away from it.

Your source code remains on your PC. A lightweight Go agent runs locally, and a mobile application sends approved Git requests through an authenticated backend relay. The agent executes Git locally and returns the result to the phone.

Pushly does not require an IDE to be open, remote desktop software, port forwarding, or uploading an entire project to the cloud.

## Project Status

Pushly is currently an early Windows-first prototype.

The local Go CLI currently supports safe, read-only Git operations:

- Repository status
- Working-tree diff
- Current branch
- Configured remotes

Repository discovery, commit and push workflows, the backend relay, and the Flutter mobile app are planned components of the product and are not implemented yet.

## How It Works

```text
Flutter mobile app
	|
     HTTPS/WSS
	|
Pushly backend relay
	|
   Authenticated outbound connection
	|
Go desktop agent
	|
Local filesystem and installed Git
	|
Git repositories on the PC
```

The desktop agent establishes the outbound connection, so users do not need to expose their computer directly to the public internet or configure port forwarding.

## Planned User Experience

1. Install the Pushly agent on a Windows PC.
2. Sign in and pair the phone with a short-lived QR code or pairing code.
3. Select parent folders that Pushly is allowed to access.
4. Discover Git repositories inside those folders.
5. Select a repository and review its status and changed files.
6. Select the files to include in a commit.
7. Review the diff and enter a commit message.
8. Commit and push using the PC's existing Git authentication.
9. Receive the operation result on the phone.

## Local CLI

### Requirements

- Go 1.26 or later
- Git installed and available on `PATH`
- A local Git repository

### Run the CLI

From the repository root:

```powershell
go run ./cmd/pushly status .
go run ./cmd/pushly diff .
go run ./cmd/pushly branch .
go run ./cmd/pushly remote .
```

The command format is:

```text
pushly <operation> <repository-path>
```

The current CLI supports only these fixed operations:

| Operation | Description |
| --- | --- |
| `status` | Shows the branch and short working-tree status. |
| `diff` | Shows the unstaged working-tree diff. |
| `branch` | Shows the current branch. |
| `remote` | Shows configured Git remotes. |

Unsupported operations are rejected. The CLI does not accept arbitrary shell commands.

## Architecture

### Go Desktop Agent

The agent is responsible for local operations:

- Connecting to the backend
- Managing authorized folders
- Discovering repositories
- Reading repository state
- Running approved Git operations
- Maintaining connection and operation state
- Returning structured results

The first target platform is Windows. Platform-specific filesystem and process behavior will be kept behind portable interfaces so macOS and Linux support can be added later.

### Backend Relay

The backend is responsible for communication and authorization:

- Account authentication
- Device registration and sessions
- Phone-to-agent pairing
- Command routing
- Authorization
- Operation metadata and history
- Notifications

The backend should relay commands and results without storing source code.

### Mobile Application

The Flutter application will provide:

- Login and account management
- Device selection
- Repository browsing
- Status and changed-file views
- Diff viewing
- Commit and push controls
- Operation progress and results

## Security Principles

Pushly is designed around a strict local-operation boundary.

- The agent uses an explicit allowlist of supported operations.
- Arbitrary shell, PowerShell, and command prompt execution is not allowed.
- Every request is checked against the user, device, repository, and operation.
- Phone-agent pairing uses a short-lived code or QR flow and supports revocation.
- The agent independently validates command scope, repository paths, expiry, and replay protection.
- Diff content is intended to be end-to-end encrypted between the agent and phone.
- The backend must not inspect, persist, or log plaintext source-code diffs.
- Git credentials and SSH private keys remain on the user's computer.
- The agent uses the computer's existing Git authentication configuration.
- Repository scans are bounded and do not follow symlinks.
- Write operations require a fresh repository status before staging or committing.
- Operations are serialized per repository to avoid conflicting local actions.

## Repository Access Rules

Users explicitly configure parent folders. The agent will only expose repositories inside those folders.

Repository discovery will:

- Enforce scan-depth and repository-count limits.
- Skip symlinks and inaccessible directories.
- Report skipped paths instead of stopping the entire scan.
- Validate repository roots using Git.
- Keep repository-relative paths separate from operating-system paths.
- Avoid reading or uploading source files during discovery.

## Planned Git Operations

The product is intended to support these operations through typed requests rather than arbitrary command strings:

### Read operations

- Status
- Branch information
- Remote information
- Commit history
- Diff inspection

### Write operations

- Stage explicitly selected files
- Commit selected files
- Push
- Pull
- Switch branches

Before a remote commit, the user must select the files to include. Pushly must never stage all changes implicitly. If the repository changed after the user reviewed it, the operation must stop and require a fresh review.

## Development Principles

- Keep the source code on the user's computer.
- Prefer the installed Git executable over reimplementing Git.
- Keep operations explicit, typed, and testable.
- Use machine-readable Git output for parsing.
- Treat untracked files separately because normal `git diff` does not include them.
- Use operation IDs and state reconciliation after uncertain network failures.
- Keep the backend out of the source-code storage path.
- Build and test the Windows agent before expanding to other platforms.

Detailed implementation tracking is maintained separately in [IMPLEMENTATION_PHASES.md](IMPLEMENTATION_PHASES.md).

## Repository Structure

```text
pushly/
├── cmd/
│   └── pushly/
│       └── main.go
├── internal/
│   └── git/
│       ├── git.go
│       └── git_test.go
├── IMPLEMENTATION_PHASES.md
├── PUSHLY_PROJECT_DESCRIPTION.md
├── Readme.md
└── go.mod
```

## Long-Term Direction

The initial product is focused on secure remote Git management. Later versions may add branch workflows, pull requests, test and build execution, development-server controls, logs, AI summaries, and multi-device management.

Those capabilities will remain explicit and permissioned. Pushly is not intended to become a remote desktop application, a general-purpose remote shell, a cloud source-code host, or a replacement for GitHub.
