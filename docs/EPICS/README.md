# EPICS: Lifecycle-Aware CPU Boost

This directory contains structured EPICS (large features) for implementing lifecycle-aware CPU boost activations in kube-startup-cpu-boost.

## Overview

The EPICS in this directory implement the design proposal from `docs/background/02_design_proposal.md`, which extends kube-startup-cpu-boost from a **startup accelerator** to a **lifecycle-aware transient boost framework**.

## EPIC Index

| EPIC | Title | Status | Dependencies |
|------|-------|--------|--------------|
| [EPIC-001](./EPIC-001-trigger-system-foundation.md) | Trigger System Foundation | Not Started | None |
| [EPIC-002](./EPIC-002-container-restart-trigger.md) | ContainerRestart Trigger | Not Started | EPIC-001 |
| [EPIC-003](./EPIC-003-pod-condition-transition-trigger.md) | PodConditionTransition Trigger | Not Started | EPIC-001 |
| [EPIC-004](./EPIC-004-cooldown-rate-limiting.md) | Cooldown and Rate Limiting | Not Started | EPIC-001 |
| [EPIC-005](./EPIC-005-observability-events.md) | Observability and Events | Not Started | EPIC-001 |
| [EPIC-006](./EPIC-006-multi-trigger-composability.md) | Multi-Trigger Composability | Not Started | EPIC-001, EPIC-002, EPIC-003 |
| [EPIC-007](./EPIC-007-backward-compatibility.md) | Backward Compatibility and Migration | Not Started | EPIC-001 |

## Implementation Order

Based on the design proposal, the recommended implementation order is:

1. **EPIC-001: Trigger System Foundation** (foundational)
   - Establishes API schema and internal refactoring
   - No behavior changes, enables future features

2. **EPIC-007: Backward Compatibility** (parallel with EPIC-001)
   - Ensures existing CRs continue working
   - Critical for safe upgrades

3. **EPIC-002: ContainerRestart Trigger** (first new feature)
   - Implements restart detection and boost application
   - Relatively straightforward compared to condition transitions

4. **EPIC-004: Cooldown and Rate Limiting** (safety feature)
   - Prevents pathological loops
   - Should be implemented before or alongside EPIC-003

5. **EPIC-003: PodConditionTransition Trigger** (complex feature)
   - Implements condition transition detection
   - Benefits from cooldown (EPIC-004)

6. **EPIC-006: Multi-Trigger Composability** (enhancement)
   - Enables combining multiple triggers
   - Requires EPIC-002 and EPIC-003 to be complete

7. **EPIC-005: Observability and Events** (cross-cutting)
   - Can be implemented incrementally
   - Enhances all other EPICS

## EPIC Structure

Each EPIC document follows this structure:

- **Summary**: High-level description
- **Business Value**: Why this matters
- **Goals**: What we're trying to achieve
- **User Stories**: Detailed stories with acceptance criteria
- **Dependencies**: Other EPICS this depends on
- **Risks**: Potential issues and mitigations
- **Success Metrics**: How we measure success
- **Implementation Order**: Suggested order for user stories
- **Related EPICS**: Cross-references
- **Test Coverage Analysis**: Current coverage status, gaps, requirements, and test strategy

## User Story Format

Each user story includes:

- **Title**: As a [role], I want [goal], So that [benefit]
- **Acceptance Criteria**: Checklist of requirements
- **Technical Notes**: Implementation guidance

## Status Tracking

Update EPIC status as work progresses:

- **Not Started**: Planning phase
- **In Progress**: Active development
- **Review**: Code review or design review
- **Testing**: Integration and E2E testing
- **Done**: Complete and merged

## Dependencies Graph

```
EPIC-001 (Foundation)
├── EPIC-002 (ContainerRestart)
├── EPIC-003 (PodConditionTransition)
├── EPIC-004 (Cooldown)
├── EPIC-005 (Observability)
└── EPIC-007 (Backward Compatibility)

EPIC-002 + EPIC-003
└── EPIC-006 (Multi-Trigger)
```

## Key Design Principles

1. **Backward Compatibility**: Existing CRs must continue working
2. **Additive Changes**: All API changes are optional/additive
3. **Explicit Triggers**: Separate "when to activate" from "when to stop"
4. **Safety First**: Cooldown prevents pathological loops
5. **Observable**: Events and metrics for all boost lifecycle events
6. **Test Coverage**: Each EPIC includes coverage analysis and test requirements

## Related Documentation

- [Design Proposal](../background/02_design_proposal.md): Detailed technical design
- [Proposed Solution](../background/01_proposed_solution.md): Problem analysis and solution approach
- [Introduction](../background/00_intro.md): Problem statement and context

## Contributing

When working on an EPIC:

1. **Create story branch** from `development`:
   ```bash
   git checkout development
   git pull origin development
   git checkout -b story/epic-{number}-us-{number}-{short-description}
   ```

2. Update the EPIC status in this README
3. Check off user stories as they're completed
4. Document any deviations from the plan
5. Update related EPICS if dependencies change
6. Add test coverage for all acceptance criteria
7. Create PR targeting `development` branch
8. After merge, delete story branch

See [Development Guide](../../DEVELOPMENT.md#branching-strategy) for detailed branching workflow.

## Questions or Issues

If you encounter issues or have questions about EPIC implementation:

1. Review the design proposal in `docs/background/`
2. Check related EPICS for context
3. Update EPIC documentation with learnings
4. Consider if EPIC scope needs adjustment

