# Branching Workflow for Lifecycle-Aware CPU Boost

This document describes the branching workflow for implementing the lifecycle-aware CPU boost EPICS.

## Overview

We use a **long-lived `development` branch** where all story branches are created from and merged back to until code complete.

## Branch Hierarchy

```
main (production-ready)
  └── development (long-lived feature branch)
       ├── story/epic-001-us-001-api-schema
       ├── story/epic-001-us-002-activation-model
       ├── story/epic-001-us-003-state-tracking
       ├── story/epic-002-us-005-restart-detection
       └── ... (other story branches)
```

## Workflow Steps

### 1. Starting a New Story

**Before starting work**:
```bash
# Ensure you're on development and up to date
git checkout development
git pull origin development

# Create story branch
git checkout -b story/epic-001-us-001-api-schema
```

**Branch naming**: `story/epic-{number}-us-{number}-{short-description}`
- `epic-001-us-001-api-schema` - EPIC 1, User Story 1, API schema work
- `epic-002-us-005-restart-detection` - EPIC 2, User Story 5, restart detection

### 2. Working on Story

**During development**:
- Make commits following [Conventional Commits](https://www.conventionalcommits.org/)
- Push branch regularly:
  ```bash
  git push -u origin story/epic-001-us-001-api-schema
  ```

**Commit message format**:
```
feat(epic-001): add triggers API schema

- Add BoostTrigger type to StartupCPUBoostSpec
- Add validation for trigger types
- Update CRD generation

Closes #<issue-number>
```

### 3. Creating Pull Request

**When story is ready**:
1. Ensure branch is up to date with `development`:
   ```bash
   git checkout development
   git pull origin development
   git checkout story/epic-001-us-001-api-schema
   git rebase development  # or merge if preferred
   ```

2. Push updated branch:
   ```bash
   git push origin story/epic-001-us-001-api-schema
   ```

3. Create PR on GitHub:
   - **Base branch**: `development`
   - **Compare branch**: `story/epic-001-us-001-api-schema`
   - **Title**: `[EPIC-001] US-001: API Schema for Triggers`
   - **Description**: Reference EPIC and user story, include acceptance criteria checklist

### 4. PR Review and Merge

**During review**:
- Address review comments
- Update PR with new commits
- Ensure all CI checks pass

**After approval**:
- Merge to `development` (squash or merge as preferred)
- Delete story branch (GitHub option or manually):
  ```bash
  git checkout development
  git pull origin development
  git branch -d story/epic-001-us-001-api-schema
  git push origin --delete story/epic-001-us-001-api-schema
  ```

### 5. Repeat Until Complete

Continue creating story branches from `development` until all EPICS are code complete.

### 6. Final Merge to Main

**When all EPICS complete**:
1. Create PR from `development` to `main`
2. Include summary of all changes
3. Ensure all tests pass, documentation updated
4. After merge, tag release

## Branch Naming Examples

| EPIC | User Story | Branch Name |
|------|------------|-------------|
| EPIC-001 | US-001 | `story/epic-001-us-001-api-schema` |
| EPIC-001 | US-002 | `story/epic-001-us-002-activation-model` |
| EPIC-001 | US-003 | `story/epic-001-us-003-state-tracking` |
| EPIC-002 | US-005 | `story/epic-002-us-005-restart-detection` |
| EPIC-002 | US-006 | `story/epic-002-us-006-container-filtering` |
| EPIC-003 | US-009 | `story/epic-003-us-009-transition-detection` |
| EPIC-004 | US-013 | `story/epic-004-us-013-cooldown-api` |
| EPIC-005 | US-018 | `story/epic-005-us-018-activation-events` |
| EPIC-006 | US-024 | `story/epic-006-us-024-multi-trigger-config` |
| EPIC-007 | US-029 | `story/epic-007-us-029-default-trigger` |

## Best Practices

### ✅ Do

- **Always branch from latest `development`**: `git checkout development && git pull`
- **Keep branches focused**: One user story per branch
- **Update EPIC status**: Mark user stories complete in EPIC docs
- **Write tests**: Include test coverage as specified in EPIC
- **Update documentation**: Keep docs in sync with code changes
- **Delete merged branches**: Keep repository clean

### ❌ Don't

- **Don't branch from other story branches**: Always use `development`
- **Don't merge story branches to `main`**: Only `development` → `main`
- **Don't skip tests**: All acceptance criteria must be tested
- **Don't leave branches open**: Delete after merge

## Handling Dependencies

**If story depends on another story**:

1. **Option 1**: Wait for dependency to merge to `development`, then branch
2. **Option 2**: If urgent, branch from dependency branch, but merge both to `development` in order

**Example**:
- EPIC-002 depends on EPIC-001
- Complete EPIC-001 stories first
- Merge EPIC-001 to `development`
- Then branch EPIC-002 stories from updated `development`

## CI/CD Considerations

**Current CI setup**:
- Build workflow runs on `main` branch and PRs to `main`
- May need to update workflows to also run on `development` branch

**Recommendation**: Update `.github/workflows/build.yaml` to include `development`:
```yaml
on:
  push:
    branches:
      - main
      - development  # Add this
  pull_request:
    branches:
      - main
      - development  # Add this
```

## Tracking Progress

**In EPIC documents**:
- Update user story checkboxes as work progresses
- Update EPIC status in README
- Note any deviations or learnings

**In PR descriptions**:
- Reference EPIC and user story numbers
- Include acceptance criteria checklist
- Link to related EPICS if applicable

## Troubleshooting

**Branch conflicts with development**:
```bash
git checkout development
git pull origin development
git checkout story/epic-001-us-001-api-schema
git rebase development
# Resolve conflicts, then:
git push --force-with-lease origin story/epic-001-us-001-api-schema
```

**Need to update from development**:
```bash
git checkout story/epic-001-us-001-api-schema
git merge development  # or git rebase development
```

**Accidentally branched from wrong branch**:
```bash
# Rebase onto development
git checkout story/epic-001-us-001-api-schema
git rebase --onto development <wrong-base-branch>
```

## Related Documentation

- [Development Guide](../../DEVELOPMENT.md) - General development practices
- [EPICS README](./README.md) - EPIC structure and status
- [ADR-001](../ADR-001-lifecycle-aware-cpu-boost.md) - Architecture decision

