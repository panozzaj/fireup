# Git Worktree Auto-Subdomains for AI Agents

## Problem

AI coding agents (Claude Code, Cursor, Copilot Workspace, etc.) increasingly work in git worktrees to handle multiple tasks in parallel. Each worktree represents a different branch with potentially different code running on the same ports. Currently there's no easy way to access each worktree's services via distinct URLs.

## Use Case

Developer has Claude Code running in multiple worktrees:

- `~/projects/myapp` (main branch)
- `~/projects/myapp-feature-123` (feature-123 branch)
- `~/projects/myapp-bugfix-456` (bugfix-456 branch)

Each worktree may have its own fireup.yaml and running services on the same logical ports. The agent (or developer) needs to access each one via distinct URLs for testing.

## Proposed Solution

Auto-generate subdomains based on branch name:

```
feature-123.myapp.test → worktree on feature-123 branch
bugfix-456.myapp.test → worktree on bugfix-456 branch
myapp.test → main/master branch (default)
```

### Detection

fireup could detect worktrees via:

1. `git worktree list` in project directories
2. Scanning for multiple fireup.yaml files that share the same `name`
3. Explicit configuration linking worktrees together

### Port Allocation

Each worktree needs its own port range. Options:

1. **Auto-offset**: Branch hash determines port offset (e.g., feature-123 → base port + 1000)
2. **Dynamic allocation**: fireup assigns ports and handles routing
3. **Explicit config**: Each worktree specifies its own ports

### Routing

When request comes to `feature-123.myapp.test`:

1. Extract branch prefix from subdomain
2. Find worktree running that branch
3. Route to that worktree's service port

## Configuration Ideas

### Option A: Auto-discovery

```yaml
# fireup.yaml
name: myapp
worktrees:
    auto: true # detect sibling worktrees automatically
    subdomain_pattern: '{branch}.{name}.test'
```

### Option B: Explicit linking

```yaml
# fireup.yaml in main worktree
name: myapp
worktrees:
    - path: ../myapp-feature-123
      branch: feature-123
    - path: ../myapp-bugfix-456
      branch: bugfix-456
```

### Option C: Independent with shared name

```yaml
# Each worktree's fireup.yaml
name: myapp
branch: feature-123 # fireup auto-routes {branch}.myapp.test here
```

## Benefits

- AI agents can work on multiple branches simultaneously with isolated environments
- Easy to test/compare different implementations side-by-side
- No port conflicts between worktrees
- Developer can check any branch's running state via predictable URL

## Open Questions

- How to handle branch names with special characters (slashes, etc.)?
- Should this integrate with the dashboard to show all worktrees?
- How to handle worktrees on same branch?
- Should fireup manage starting/stopping services across worktrees, or just routing?

## Related

- Git worktree docs: https://git-scm.com/docs/git-worktree
- Common pattern for AI agents working in parallel on different features
