## Branch naming

### branch.pattern

Branch naming pattern

**Placeholders: {username}, {date}, {message}, {scope}**

**Default**: (not set)

**Example**:

```bash
stackit config set branch.pattern "{username}/{date}/{message}"
```

## PR submission

### submit.footer

Include navigation footer

**Default**: `true`

**Example**:

```bash
stackit config set submit.footer false
```

### submit.draft

Create as draft

**Default**: `false`

**Example**:

```bash
stackit config set submit.draft true
```

### submit.web

Open in browser: always, created, never

**Default**: `never`

**Options**: `always`, `created`, `never`

**Example**:

```bash
stackit config set submit.web never
```

### submit.labels

Default labels

### submit.reviewers

Default reviewers

### submit.assignees

Default assignees

## PR navigation

### navigation.when

always, never, multiple

**Default**: `multiple`

**Options**: `always`, `never`, `multiple`

**Example**:

```bash
stackit config set navigation.when multiple
```

### navigation.marker

Current branch marker

**Default**: `👈`

**Example**:

```bash
stackit config set navigation.marker 👈
```

### navigation.location

body, comment, none

**Default**: `body`

**Options**: `body`, `comment`, `none`

**Example**:

```bash
stackit config set navigation.location body
```

### navigation.showMerged

Show merged history

**Default**: `true`

**Example**:

```bash
stackit config set navigation.showMerged false
```

## Merge settings

### merge.method

Merge method: squash, merge, rebase

**Default**: (not set)

**Options**: `squash`, `merge`, `rebase`

**Example**:

```bash
stackit config set merge.method squash
```

## CI validation

### ci.command

Command to run

**Default**: (not set)

**Example**:

```bash
stackit config set ci.command "make test"
```

### ci.timeout

Timeout in seconds

**Default**: `600`

**Example**:

```bash
stackit config set ci.timeout 600
```

## Worktree settings

### worktree.basePath

Base directory (empty = auto)

### worktree.autoClean

Clean during sync

**Default**: `true`

**Example**:

```bash
stackit config set worktree.autoClean false
```

## Split command

### split.hunkSelector

tui or git

**Default**: `tui`

**Options**: `tui`, `git`

**Example**:

```bash
stackit config set split.hunkSelector tui
```

## Trunk branches

### trunk

Primary trunk branch

**Default**: `main`

**Example**:

```bash
stackit config set trunk main
```

### trunks

Additional trunk branches (e.g., release branches)

**Default**: (not set)

**Example**:

```bash
stackit config set trunks "develop, release"
```

## Other settings

### undo.depth

Max snapshots

**Default**: `10`

**Example**:

```bash
stackit config set undo.depth 10
```

### maxConcurrency

0 = auto-detect

**Default**: `0`

**Example**:

```bash
stackit config set maxConcurrency 0
```


## Team configuration (`.stackit.yaml`)

For team-wide settings that should be shared across all contributors, create a `.stackit.yaml` file in your repository root and commit it to version control. Team settings act as defaults that individual developers can override in their personal git config.

See the [Team Collaboration Guide](../workflows/collaboration.md) for collaboration patterns using shared configuration.

```yaml
# .stackit.yaml - Team-wide defaults

trunk: main
trunks:
  - develop
  - staging

# Branch naming pattern for the team
branch:
  pattern: "{username}/{date}/{message}"

# PR submission settings
submit:
  footer: true
  draft: false
  web: never
  labels:
    - needs-review
  reviewers:
    - teammate1
  assignees: []

# Default merge method
merge:
  method: squash

# CI validation
ci:
  command: "make test"
  timeout: 600

# Undo history
undo:
  depth: 10

# Worktree settings
worktree:
  basePath: ""
  autoClean: true

# Split settings
split:
  hunkSelector: tui

# Concurrency (0 = auto based on CPU count)
maxConcurrency: 0

# PR navigation display options
navigation:
  when: multiple
  location: body
  marker: 👈
  showMerged: true

# Worktree hooks
hooks:
  post-worktree-create:
    - npm install
    - mise install

```
