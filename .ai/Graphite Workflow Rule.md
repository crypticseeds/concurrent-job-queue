---
apply: manually
---

---

## trigger: manual

# Graphite Workflow Rule — Safe Stack Development

## Purpose

This rule defines how branches, commits, and stacks are managed using Graphite (`gt`).

The goal is to maintain:

* clean stacked branches
* safe history management
* predictable CI and PR workflows
* controlled commit submission after verification

---

# Core Principles

### Verification Before Commit

**Do not commit changes until verification is complete.**

Requirements before creating a commit:

* User has reviewed the implementation
* Tests have passed
* Linting has passed
* The user explicitly confirms the changes are correct

Only after verification may a commit be created.

Commit creation must use:

```
gt create <branch-name>
```

Standard git commits should not be used.

---

# Strictly Prohibited Commands

The following commands must **never be executed**:

```
gt ss
gt submit
gt squash
```

Pull request submission and squash operations are handled manually outside this workflow.

---

# Branch Navigation

Use Graphite commands to move through stacked branches.

| Command     | Action                                        |
| ----------- | --------------------------------------------- |
| `gt up`     | Move to the branch above the current one      |
| `gt down`   | Move to the parent branch                     |
| `gt prev`   | Move to the previous branch in stack order    |
| `gt next`   | Move to the next branch in stack order        |
| `gt top`    | Move to the tip of the stack                  |
| `gt bottom` | Move to the branch closest to trunk           |
| `gt ls`     | Display all tracked branches and dependencies |

These commands should be used instead of raw Git navigation when working within a stack.

---

# Branch Creation Workflow

When beginning implementation for an issue:

1. Ensure you are on the correct parent branch.
2. Create a stacked branch.

Example:

```
gt create issue-worker-pool
```

Branch naming should follow:

```
issue-<short-description>
```

Examples:

```
issue-worker-pool
issue-http-api
issue-graceful-shutdown
```

---

# Commit Management

All commits must use Graphite commands.

Priotised stacked branch when committing changes over modify current branch to add changes unless fixing a conflict.

Preferred commands:

Create commit:

```
gt modify -c -m "message"
```

Amend commit:

```
gt modify --amend
```

Stage and commit all changes:

```
gt modify -cam "message"
```

Never use:

```
git commit
git commit --amend
```

These bypass Graphite’s stack tracking.

---

# Stack Maintenance

To maintain stack integrity: Don't use this during commit, only when asked or solving commit issues that require restack.

```
gt restack
```

Purpose:

* ensure each branch correctly references its parent
* automatically rebase descendants when needed

Important:

`gt restack` operates **locally only** and does not interact with remote repositories.

---

# Syncing with Remote

To update branches from remote:

```
gt get
```

To sync all branches:

```
gt sync
```

These commands ensure the stack remains aligned with the remote repository.

---

# Handling Interactive Commands

Some Graphite commands are interactive.

Examples:

* `gt modify`
* `gt move`
* `gt split`
* `gt restack`

Rules:

* Prefer non-interactive flags where possible
* If user input is required and cannot be automated, stop and request guidance.

---

# Conflict Resolution Workflow

If a restack or move results in merge conflicts:

1. Identify conflicts:

```
git status
```

2. Resolve merge markers:

```
<<<<<<<
=======
>>>>>>>
```

3. Stage resolved files:

```
gt add -A
```

4. Continue the process:

```
gt continue
```

Important:

Never run:

```
git rebase --continue
```

Graphite maintains its own internal state.

---

# Misplaced Commit Recovery

If a commit is added to the wrong branch:

Move the commit safely using Graphite.

Example approach:

1. Create a new branch containing the commit:

```
gt create <new-branch>
```

2. Move back to parent:

```
gt down
```

3. Reset parent branch:

```
git reset --hard HEAD~1
```

4. Repair stack:

```
gt restack
```

---

# Expected Workflow Summary

Correct workflow:

1. Navigate to correct branch
2. Create stacked branch with `gt create`
3. Implement changes locally
4. Run tests and linting
5. Wait for **user verification**
6. Create commit using `gt modify`
7. Maintain stack using `gt restack`

---

# Safety Requirements

Before committing changes ensure:

* code compiles
* tests pass
* lint checks pass
* user verification is complete

No commits should be created automatically before these conditions are met.
