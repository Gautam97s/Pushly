# Pushly Implementation Phases

This document is the detailed implementation tracker for Pushly. The project is developed in small, testable slices. Each phase should be completed and verified before the next phase expands the system.

## Status Legend

- **Completed**: implemented and verified.
- **In progress**: currently being implemented.
- **Planned**: defined but not started.
- **Deferred**: intentionally postponed until the MVP is stable.

## Overall MVP Goal

A developer should be able to leave their IDE closed, open Pushly on a phone from another network, select a repository that remains on a Windows PC, review its changes, select files, commit, push to GitHub using the PC's existing Git authentication, and receive a reliable result.

---

## Phase 1: Local Git Core

**Status: Completed**

### Purpose

Create a small local CLI and a safe Git execution boundary before adding scanning, networking, or mobile features.

### Implemented parts

- Initialized the Go module.
- Added the `pushly` CLI entry point.
- Added a Git client using the system Git executable through Go's `os/exec`.
- Added a fixed operation allowlist.
- Added read-only operations:
  - `status`
  - `diff`
  - `branch`
  - `remote`
- Added repository working-directory support.
- Added command timeouts.
- Added combined Git output and structured execution errors.
- Added unit tests for allowed and rejected operations.

### Current commands

```powershell
go run ./cmd/pushly status .
go run ./cmd/pushly diff .
go run ./cmd/pushly branch .
go run ./cmd/pushly remote .
```

### Acceptance criteria

- [x] Go module builds successfully.
- [x] Read-only commands execute against a real repository.
- [x] Unsupported operations are rejected.
- [x] Git command failures are returned to the user.
- [x] Unit tests pass.
- [x] The CLI does not accept arbitrary shell commands.

### Known limitations

- Git output is still returned as raw text.
- `git diff` does not include untracked files.
- There are no write operations yet.
- Repository paths are not yet validated against configured folders.

---

## Phase 2: Repository and Scanner Layer

**Status: In progress**

### Purpose

Discover authorized repositories and convert Git output into structured data that the agent and mobile app can safely use.

### Part 2.1: Structured Git status

- Add a machine-readable status operation using Git porcelain output.
- Prefer NUL-separated records for reliable path parsing.
- Parse the current branch and upstream information.
- Parse ahead and behind counts.
- Represent repository cleanliness.
- Represent each change with:
  - Repository-relative path.
  - Original path for renames.
  - Staged state.
  - Worktree state.
  - Untracked state.
  - Modified, added, deleted, or renamed state.
- Preserve spaces, Unicode characters, and rename pairs.

### Part 2.2: Repository validation

- Identify the real repository root using Git.
- Reject paths that are not repositories.
- Normalize Windows paths before comparison.
- Ensure requested repositories remain inside configured parent folders.
- Prevent path traversal and symlink escapes.
- Keep repository-relative paths separate from operating-system paths.

### Part 2.3: Configured folders

- Store configured parent directories.
- Normalize and validate directories when they are registered.
- Prevent duplicate or overlapping folder registrations where appropriate.
- Report inaccessible configured folders clearly.

### Part 2.4: Bounded repository discovery

- Recursively scan configured folders.
- Detect `.git` directories and worktree files.
- Enforce maximum scan depth.
- Enforce maximum repository count.
- Skip symlinks.
- Skip inaccessible directories and continue scanning.
- Record skipped paths and reasons.
- Support explicit rescans.
- Do not read or upload source-file contents during discovery.

### Part 2.5: Repository metadata and snapshots

Store or calculate:

- Repository name.
- Repository root.
- Current branch.
- Remote names and URLs, with credentials removed.
- Last scan time.
- Current status snapshot.
- Snapshot identifier or hash for stale-state detection.

Snapshots must be immutable. Before a future write operation, the agent will compare the latest repository state with the reviewed snapshot.

### Part 2.6: Untracked-file behavior

Normal `git diff` does not include untracked files. Pushly will therefore:

- Show untracked files explicitly in status.
- Avoid reading untracked content during repository discovery.
- Provide controlled content or diff access only after the user selects a file.
- Validate selected files against the repository root before reading them.
- Avoid sending source content to the backend except as an encrypted user-requested payload.

### Acceptance criteria

- [ ] Parse clean repositories.
- [ ] Parse modified, staged, untracked, deleted, and renamed files.
- [ ] Parse paths containing spaces and Unicode characters.
- [ ] Parse rename pairs correctly.
- [ ] Discover multiple repositories in nested folders.
- [ ] Respect scan depth and repository-count limits.
- [ ] Skip symlinks and report inaccessible paths.
- [ ] Reject repositories outside configured folders.
- [ ] Prevent path traversal and symlink escapes.
- [ ] Confirm discovery does not upload source code.
- [ ] Detect when a status snapshot is stale.
- [ ] Add unit tests and Windows filesystem integration tests.

---

## Phase 3: Local Windows Agent

**Status: Planned**

### Purpose

Turn the local Git core and repository layer into a Windows background agent that can safely perform approved write operations.

### Parts

- Add Windows configuration storage.
- Add explicit changed-file selection.
- Validate every selected repository-relative path.
- Stage only selected files.
- Never stage all changes implicitly.
- Add commit operation with commit-message validation.
- Add push operation using existing local Git authentication.
- Run Git hooks normally.
- Enforce timeouts and bounded output.
- Prevent interactive Git prompts from hanging the agent.
- Serialize operations per repository.
- Refresh status immediately before write operations.
- Reject stale status snapshots.
- Add structured operation states and results.
- Add Windows background-process or service support.
- Add graceful shutdown and reconnect-ready lifecycle handling.

### Acceptance criteria

- [ ] A user can select specific files for a commit.
- [ ] Unselected changes are not committed.
- [ ] Invalid paths are rejected.
- [ ] A stale snapshot blocks the write operation.
- [ ] Commit and push use local Git credentials without Pushly storing them.
- [ ] Hook failures are returned clearly.
- [ ] One repository cannot run conflicting operations concurrently.
- [ ] The agent can run without an IDE being open.

---

## Phase 4: Secure Backend and Agent Protocol

**Status: Planned**

### Purpose

Connect the mobile app and desktop agent through an authenticated cloud relay without exposing the PC directly to the public internet.

### Parts

- Add Pushly account authentication.
- Register and authenticate desktop devices.
- Use outbound WSS from the agent.
- Add short-lived QR/code pairing.
- Bind device keys after pairing.
- Support pairing revocation.
- Add authenticated phone sessions.
- Define typed command and result messages.
- Include operation ID, expiry, nonce, repository identity, operation type, and validated arguments.
- Require independent agent-side authorization.
- Enforce the operation allowlist at the agent.
- Add replay protection.
- Add per-user, per-device, per-repository, and per-operation authorization.
- Relay encrypted diff payloads without reading or storing plaintext source content.
- Store operation metadata without storing source code or credentials.
- Handle agent reconnects and heartbeats.
- Reconcile state after uncertain network failures.
- Never blindly repeat commit or push commands.

### Acceptance criteria

- [ ] The agent establishes an outbound authenticated connection.
- [ ] A phone can pair through a short-lived code or QR code.
- [ ] Revoked devices cannot issue commands.
- [ ] Expired and replayed commands are rejected.
- [ ] Invalid repository and path scopes are rejected by the agent.
- [ ] Arbitrary shell commands are rejected.
- [ ] The backend cannot read encrypted diff payloads.
- [ ] Disconnects do not create duplicate commits or pushes.

---

## Phase 5: Flutter Mobile MVP

**Status: Planned**

### Purpose

Provide the user-facing mobile workflow for controlling the authorized Windows agent.

### Parts

- Login and account session.
- Device list and device status.
- Pairing flow.
- Configured-folder and repository list.
- Repository details.
- Branch and status display.
- Changed-file selection.
- Encrypted diff viewing.
- Commit-message input.
- Commit action.
- Push action.
- Operation progress.
- Success and failure results.
- Operation history and notifications.
- Actionable messages for stale state, conflicts, authentication errors, hooks, timeouts, and offline agents.

### Acceptance criteria

- [ ] A user can pair with an authorized desktop agent.
- [ ] A user can select a repository.
- [ ] Status and changed files are displayed accurately.
- [ ] A user can select exactly which files to commit.
- [ ] Diffs are displayed after encrypted transfer.
- [ ] Commit and push results are reliable and understandable.
- [ ] Sensitive source content is not persisted by the backend.

---

## Phase 6: Security and Release Hardening

**Status: Planned**

### Purpose

Prepare Pushly for real users and hostile or unreliable environments.

### Parts

- Review token storage and expiry.
- Review device revocation and account recovery.
- Test path traversal and symlink escapes.
- Test unauthorized devices and repositories.
- Test replayed and expired commands.
- Test arbitrary command injection attempts.
- Test oversized command and Git output.
- Test Git hook timeouts.
- Test malformed Git output and unusual filenames.
- Test concurrent local Git activity.
- Test disconnects during commit and push.
- Add rate limiting.
- Add audit logging without sensitive payloads.
- Add secure Windows installation and update handling.
- Document Git prerequisites and authentication behavior.
- Add portable abstractions for later macOS and Linux support.

### Release gate

Pushly is ready for MVP release only when a Windows-first end-to-end test succeeds with:

1. IDE closed.
2. Agent running in the background.
3. Phone on another network.
4. Authorized repository discovered locally.
5. Status reviewed on the phone.
6. Explicit files selected.
7. Commit created locally.
8. Push completed through existing Git authentication.
9. Reliable result returned to the phone.
10. No source code or credentials stored by the backend.

---

## Future Features

These remain outside the MVP phases until the release gate is complete:

- Pull and branch switching.
- Branch creation, merge, stash, tags, and revert.
- Pull request creation.
- Test and build execution.
- Development server control.
- Log streaming.
- AI-generated commit messages and summaries.
- AI code review.
- Multi-device management beyond the initial pairing model.
- macOS and Linux service packaging.
