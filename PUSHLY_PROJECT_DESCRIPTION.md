# Pushly

> **Remotely push and manage your local Git repositories from anywhere.**

## 1. Project Overview

**Pushly** is a cross-platform remote Git management system that allows a developer to control Git repositories stored on their personal computer through a mobile application.

The core idea is simple:

- The developer's code remains on their PC.
- A lightweight **Go-based desktop agent** runs in the background.
- The developer uses a **mobile app** to discover and select repositories on the PC.
- The mobile app sends commands securely through a cloud relay/backend.
- The Go agent executes the requested Git operation locally.
- Results are returned to the mobile app.

Pushly is designed to work even when the repository is **not currently open in VS Code, Cursor, IntelliJ, or any other IDE**. An IDE does not need to be running because Pushly interacts directly with the filesystem and the Git executable.

---

## 2. Problem Statement

Developers often have multiple repositories distributed across their computer:

```text
C:/Projects/
├── AI-Agent/
├── CollegeSocial/
├── SafePick/
└── Personal/
```

If a developer is away from their computer and wants to push changes that already exist locally, they normally need direct access to the machine or a remote development environment.

Pushly provides a lightweight remote-control layer over the developer's local Git repositories.

The developer can use their phone to:

1. Connect to their registered computer.
2. Browse available project folders.
3. Select a Git repository.
4. Inspect its Git status.
5. Review changed files.
6. Create a commit.
7. Push the commit to GitHub.
8. Pull changes when required.
9. Manage branches.
10. View operation logs and results.

---

## 3. Core Product Concept

Pushly consists of three primary components:

```text
                 ┌──────────────────┐
                 │   Mobile App     │
                 │    Flutter       │
                 └────────┬─────────┘
                          │
                    HTTPS / WebSocket
                          │
                 ┌────────▼─────────┐
                 │ Pushly Backend   │
                 │ Cloud Relay/API  │
                 └────────┬─────────┘
                          │
                    Secure connection
                          │
                 ┌────────▼─────────┐
                 │ Pushly Agent     │
                 │      Go          │
                 └────────┬─────────┘
                          │
                    Local filesystem
                          │
                 ┌────────▼─────────┐
                 │ Git Repositories │
                 │       PC         │
                 └──────────────────┘
```

### Mobile App

The mobile application is the user's control interface.

It should allow the user to:

- Register/login.
- View connected computers.
- Browse configured folders.
- Discover Git repositories.
- View repository status.
- View changed/untracked files.
- View Git diffs.
- Commit changes.
- Push changes.
- Pull changes.
- Switch branches.
- View recent commits.
- View command logs.
- Receive success/failure notifications.

### Go Desktop Agent

The desktop agent is the core local component.

It runs in the background on the user's computer and is responsible for:

- Connecting to the Pushly backend.
- Authenticating the computer.
- Managing configured folders.
- Discovering Git repositories.
- Executing Git commands.
- Reading repository state.
- Returning command results.
- Watching for repository changes.
- Maintaining a persistent connection to the backend.
- Handling reconnects.
- Logging operations.

The agent must not depend on an IDE being open.

### Backend

The backend acts as a secure relay and API layer between the mobile application and desktop agent.

Responsibilities include:

- User authentication.
- Device registration.
- Device/session management.
- WebSocket connections.
- Command routing.
- Authorization.
- Repository metadata.
- Operation history.
- Notifications.
- Security controls.

The backend should **not** need to store the user's source code.

---

## 4. Primary User Flow

### Initial Setup

1. User installs Pushly Desktop Agent.
2. User logs into their Pushly account.
3. The agent registers the computer.
4. User selects one or more parent directories to make available to Pushly.
5. Pushly performs a bounded scan of those directories for Git repositories. Symlinks and inaccessible paths are skipped and reported.
6. Repositories are displayed in the mobile app.

Example:

```text
Pushly Desktop

Configured folders:

✓ D:/Projects
✓ D:/Work
✓ C:/Users/Gautam/Development
```

The agent can discover:

```text
User selects specific files to include
  ↓
D:/Projects/AI-Agent
D:/Projects/SafePick
D:/Work/CollegeSocial
```
Agent refreshes status and validates the selected files
  ↓
Agent stages only the selected files and executes Git commit
---

## 5. Remote Push Flow

The main workflow is:

```text
User opens Pushly
        ↓
Selects computer
        ↓
Selects repository
        ↓
Pushly requests Git status
        ↓
Agent executes git status
        ↓
Agent returns status
        ↓
User reviews changes
        ↓
User enters commit message
        ↓
Pushly requests commit
        ↓
Agent executes Git commit
        ↓
Pushly requests push
        ↓
Agent executes git push
        ↓
GitHub receives commit
        ↓
Agent returns result
        ↓
Mobile shows success/failure
```

Example mobile interface:

```text
CollegeSocial

Branch
main

Changes
8 modified
2 untracked

[ View Changes ]

Commit message:
____________________________

[ Commit & Push ]
```

---

## 6. Repository Discovery

Pushly should not require an IDE to identify a project.

The Go agent can recursively inspect user-selected directories and detect Git repositories by looking for `.git` directories. Discovery is bounded by configurable depth and repository-count limits. The scanner does not follow symlinks, skips inaccessible paths, and reports skipped paths to the user.

Example:

```text
D:/Projects
│
├── AI-Agent
│   └── .git
│
├── SafePick
│   └── .git
│
├── FlutterApps
│   ├── App1
│   │   └── .git
│   └── App2
│       └── .git
│
└── Notes
```

Pushly should identify:

```text
AI-Agent
SafePick
App1
App2
```

It should not upload the source code merely to discover repositories. Users can request another scan after changing their configured folders.

---

## 7. Git Operations

The initial version should support:

### Read operations

```bash
git status
git branch --show-current
git remote -v
git log
git diff
```

### Write operations

```bash
git add
git commit
git push
git pull
git checkout
```

For remote commits, the mobile user must explicitly select the changed files to include. The agent must validate that every selected path is repository-relative and remains inside the authorized repository before running `git add`. Pushly must never stage all changes implicitly.

Each repository allows only one operation at a time. Before staging, committing, or pushing, the agent refreshes the repository status. If the repository changed since the user's review, the operation stops and the user must review the new state.

The first implementation should use the system's installed Git executable through Go's `os/exec` package rather than reimplementing Git.

The MVP uses the computer's existing Git authentication configuration, including SSH keys, credential helpers, and configured remotes. Pushly must not store GitHub passwords, SSH private keys, or GitHub tokens.

Git hooks run normally for commit and push operations. The agent must enforce operation timeouts, bounded output, and non-interactive execution. Hook failures are returned as structured operation failures.

Example conceptual flow:

```text
Go Agent
   ↓
os/exec
   ↓
git status
   ↓
stdout/stderr
   ↓
Pushly Backend
   ↓
Mobile App
```

A pure-Go Git implementation can be considered later if there is a concrete requirement for it.

---

## 8. Security Model

Security is one of the most important parts of Pushly.

The system must **never expose the user's PC directly to the public internet** if it can avoid doing so.

Recommended architecture:

```text
Phone
  │
  │ HTTPS / WSS
  ▼
Pushly Cloud
  │
  │ Secure persistent connection
  ▼
PC Agent
```

The Go agent establishes an outbound connection to the backend.

This avoids requiring users to configure:

- Port forwarding
- Public IP addresses
- Router settings
- Firewall rules

### Authentication

The system should support:

- Account authentication.
- Device authentication.
- Short-lived access tokens.
- Refresh tokens where appropriate.
- Per-device credentials.
- Secure token storage.

### Authorization

Every command should verify:

```text
User
 ↓
Device
 ↓
Repository
 ↓
Operation
```

A user must only be able to control devices and repositories they are authorized to access.

The agent independently verifies every command before execution, including the authenticated user and device, allowed operation, repository and path scope, command expiry, and replay protection. The backend must not be treated as an unrestricted command authority.

Phone-to-agent pairing uses a short-lived QR code or pairing code. After pairing, the devices bind authenticated keys, and the user can revoke the pairing. Revoked or expired devices cannot issue commands.

### Sensitive Data

Pushly should avoid storing:

- Source code
- Git credentials
- SSH private keys
- GitHub passwords

Diff content is end-to-end encrypted between the agent and the mobile app. The backend only routes the encrypted payload and must not inspect, persist, or log source-code diffs.

The desktop agent should use the user's existing Git authentication configuration where possible.

---

## 9. Network Architecture

Pushly requires reliable communication between a phone and a computer that may be behind NAT.

A persistent outbound connection from the Go agent is recommended.

```text
Go Agent
   │
   │ WSS
   ▼
Pushly Relay
   ▲
   │ HTTPS/WSS
   │
Mobile App
```

When the user presses **Push**:

```text
Mobile
  ↓
Backend
  ↓
Authenticated Agent
  ↓
Local Git
  ↓
GitHub
  ↓
Agent
  ↓
Backend
  ↓
Mobile
```

The backend primarily routes commands and results rather than storing project files.

Every command includes an operation ID, expiry, nonce, repository identity, operation type, and validated arguments. If the connection fails after an operation may have completed, the agent and backend reconcile repository state before allowing a retry. Pushly must never blindly repeat a commit or push command.

---

## 10. Go Desktop Agent

Go is the primary technology for the desktop component.

Reasons for using Go:

- Native compiled binaries.
- Cross-platform support.
- Low resource usage.
- Strong concurrency support.
- Good networking libraries.
- Simple deployment.
- Easy interaction with operating-system processes.
- Suitable for long-running background services.

Target platforms:

```text
Windows
macOS
Linux
```

The first implementation and release target is Windows. Path handling, process execution, credential behavior, and background service installation should be tested on Windows first while keeping platform-specific code behind portable interfaces for later macOS and Linux support.

The agent should eventually be distributed as a native executable/service for each platform.

---

## 11. Proposed Go Project Structure

Initial structure:

```text
pushly/
├── cmd/
│   └── pushly/
│       └── main.go
│
├── internal/
│   ├── agent/
│   ├── config/
│   ├── git/
│   ├── repository/
│   ├── scanner/
│   ├── websocket/
│   ├── auth/
│   ├── logger/
│   └── system/
│
├── pkg/
│   └── models/
│
├── configs/
│
├── tests/
│
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

The exact structure should evolve as the project grows rather than being over-engineered from day one.

---

## 12. Development Phases

### Phase 1 — Go Fundamentals

Learn enough Go to work productively:

- Variables
- Functions
- Structs
- Methods
- Packages
- Interfaces
- Error handling
- Pointers
- Slices and maps
- JSON
- File operations

Do not attempt to master Go before starting the project.

---

### Phase 2 — Pushly CLI

Create a command-line version first.

Example:

```bash
pushly scan D:/Projects
pushly status D:/Projects/SafePick
pushly diff D:/Projects/SafePick
pushly commit D:/Projects/SafePick
pushly push D:/Projects/SafePick
```

The CLI establishes the core Git functionality before networking is introduced.

The CLI must already enforce selected-file staging, repository-relative path validation, operation timeouts, non-interactive Git execution, and structured results.

---

### Phase 3 — Repository Manager

Implement:

- Folder registration.
- Recursive repository discovery.
- Repository metadata.
- Git status parsing.
- Current branch detection.
- Remote detection.
- Basic diff retrieval.
- Bounded scanning with symlink avoidance, inaccessible-path reporting, and explicit rescans.

---

### Phase 4 — Git Operations

Implement:

- Commit.
- Push.
- Pull.
- Branch switching.
- Recent commits.
- Error handling.
- Command timeouts.
- Per-repository operation serialization.
- Fresh-status checks before write operations.
- Operation IDs and state reconciliation after uncertain network failures.

Operations must return structured results rather than raw terminal output whenever possible.

Example:

```json
{
  "success": true,
  "repository": "SafePick",
  "branch": "main",
  "operation": "push",
  "message": "Push completed successfully"
}
```

---

### Phase 5 — Backend

Create the Pushly backend.

Responsibilities:

```text
Authentication
Device management
WebSocket relay
Command routing
Authorization
Operation history
```

The backend may store operation metadata and results, but must not store source code, plaintext diffs, Git credentials, or secrets. Encrypted diff payloads are opaque to the backend.

A Go backend is a strong option because it allows the backend and desktop agent to share the same language and potentially common data models.

---

### Phase 6 — Agent Networking

The desktop agent maintains a persistent authenticated WebSocket connection.

It should handle:

- Connection establishment.
- Authentication.
- Heartbeats.
- Reconnection.
- Command reception.
- Command execution.
- Result transmission.
- Graceful shutdown.
- Short-lived QR/code pairing, device-key binding, and revocation.
- Independent command authorization and replay protection.

---

### Phase 7 — Mobile Application

Build the Flutter application.

Main screens:

```text
Login
  ↓
Devices
  ↓
Repositories
  ↓
Repository Details
  ↓
Changes / Diff
  ↓
Commit & Push
```

---

### Phase 8 — Security Hardening

Before public release:

- Token security.
- Device authorization.
- TLS/WSS.
- Command validation.
- Path validation.
- Repository access restrictions.
- Rate limiting.
- Audit logs.
- Secure credential handling.
- Protection against arbitrary command execution.
- End-to-end encrypted diff transfer without backend persistence.
- Operation timeouts, bounded output, and non-interactive Git execution.
- Pairing revocation and agent-side authorization checks.
- Replay protection and operation-state reconciliation.

A major security rule:

> The backend must never be allowed to execute arbitrary shell commands on the user's machine.

The agent should expose an explicit allowlist of supported operations.

For example:

```text
Allowed:
✓ git status
✓ git diff
✓ git add
✓ git commit
✓ git push
✓ git pull

Not allowed:
✗ arbitrary shell commands
✗ powershell <anything>
✗ cmd.exe <anything>
```

This is critical because Pushly effectively becomes a remote-control system for a user's computer.

---

## 13. Future Features

Once the core system works, Pushly can expand beyond Git push.

### Git Features

- Branch creation.
- Branch switching.
- Merge.
- Stash.
- Tags.
- Commit history.
- Revert.
- Remote management.
- Pull request creation.

### Development Features

- Run tests.
- View test output.
- View build output.
- Start/stop development servers.
- View application logs.
- Restart selected services.

### AI Features

- AI-generated commit messages.
- Commit summaries.
- Diff explanations.
- Code change summaries.
- Pull request descriptions.
- Basic code review.
- Detect potentially risky changes.

### Multi-device

A user could connect:

```text
My PC
Work Laptop
Home Desktop
MacBook
```

and select which computer should execute an operation.

---

## 14. Non-Goals for Version 1

Pushly V1 should **not** attempt to become:

- A full IDE.
- A cloud code storage service.
- A complete GitHub replacement.
- A remote desktop application.
- A general-purpose remote shell.
- A source-code hosting platform.

The initial product should remain focused:

> **Securely control local Git repositories from a mobile device.**

---

## 15. Recommended Technology Stack

### Desktop Agent

```text
Language: Go
Git: System Git executable
Networking: HTTP + WebSocket
Filesystem: Go standard library
Configuration: JSON/TOML/YAML or similar
Logging: log/slog
```

### Backend

```text
Language: Go
API: net/http or a lightweight Go framework
Realtime: WebSocket
Database: PostgreSQL
Cache/temporary state: Redis if required
```

### Mobile

```text
Framework: Flutter
Language: Dart
```

### Infrastructure

```text
Docker
PostgreSQL
TLS
Cloud/VPS
CI/CD
```

The exact cloud provider should be decided after the local prototype is working.

---

## 16. MVP Definition

The first usable Pushly MVP should support exactly this:

```text
1. Install Go agent on a Windows PC
2. Log in
3. Select a parent folder
4. Discover Git repositories with a bounded scan
5. Pair and connect the phone to the PC
6. Select a repository
7. View Git status and changed files
8. Select the files to include
9. Review the encrypted diff and enter a commit message
10. Commit the selected files
11. Push to GitHub using the PC's existing Git authentication
12. See the result on the phone
```

If these twelve steps work reliably, Pushly has a valid first version.

Everything else comes afterward.

---

## 17. Success Criteria

Pushly V1 is successful when a developer can:

> Leave their IDE closed, leave their project untouched on their PC, open Pushly on their phone from another network, select a local Git repository, commit changes already present on the PC, push those changes to GitHub, and receive a reliable success/failure result.

The system should accomplish this without requiring:

- IDE access.
- Remote desktop.
- Manual router configuration.
- Port forwarding.
- Uploading the entire project to Pushly's servers.

---

## 18. Development Philosophy

Pushly should be developed incrementally.

The recommended order is:

```text
Go fundamentals
      ↓
Local Git operations
      ↓
Repository discovery
      ↓
CLI
      ↓
Desktop agent
      ↓
Backend
      ↓
Secure WebSocket
      ↓
Mobile app
      ↓
Remote push
      ↓
Security hardening
      ↓
Advanced features
```

AI tools can be used heavily during development, but generated code must be understood, tested, reviewed, and adapted rather than blindly copied.

The goal is not merely to make Pushly work.

The goal is to build a reliable developer tool while gaining practical experience in:

- Go
- Systems programming
- Networking
- Git
- WebSockets
- Authentication
- Distributed systems
- Cross-platform application development
- Mobile development
- Production security

---

## 19. Project Vision

**Pushly** starts as a remote Git push utility and can evolve into a secure mobile control plane for a developer's local development environment.

The long-term vision is:

```text
                 Pushly
                    │
       ┌────────────┼────────────┐
       │            │            │
      Git        Development      AI
       │            │            │
   Push/Pull     Tests/Logs    Summaries
   Branches      Builds        Reviews
   Commits       Services      Commits
       │            │            │
       └────────────┼────────────┘
                    │
              Developer's PC
```

The core principle remains unchanged:

> **Your code stays on your machine. Pushly gives you secure remote control over the development operations you explicitly allow.**
