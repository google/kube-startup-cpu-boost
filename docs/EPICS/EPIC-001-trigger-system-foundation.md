# EPIC-001: Trigger System Foundation

## Summary

Introduce a trigger-based activation model that separates "when to activate" (triggers) from "when to stop" (durationPolicy). This foundational EPIC establishes the API schema and internal refactoring needed to support lifecycle-aware CPU boosting.

## Business Value

- Enables repeatable CPU boost activations beyond initial pod creation
- Provides foundation for recovery acceleration features
- Maintains backward compatibility with existing deployments
- Establishes clean API boundaries for future trigger types

## Goals

- Add `spec.triggers[]` API field with default behavior (PodCreate)
- Refactor internal boost activation logic to use trigger model
- Maintain 100% backward compatibility with existing CRs
- Establish state tracking infrastructure for trigger detection

## User Stories

### US-001: API Schema for Triggers
**As a** cluster administrator  
**I want** to define when CPU boost should activate via explicit triggers  
**So that** I can control boost behavior declaratively

**Acceptance Criteria:**
- [ ] `spec.triggers[]` field added to `StartupCPUBoostSpec`
- [ ] `BoostTrigger` type with `type` field (enum: PodCreate, ContainerRestart, PodConditionTransition)
- [ ] Default behavior: if `triggers` is nil/empty, default to `[{type: PodCreate}]`
- [ ] OpenAPI schema validation for trigger types
- [ ] CRD generation includes new fields

**Technical Notes:**
- Use kubebuilder markers for validation
- Ensure CRD backward compatibility
- Add to `api/v1alpha1/startupcpuboost_types.go`

---

### US-002: Internal Activation Model Refactor
**As a** developer  
**I want** boost activation logic separated from admission webhook  
**So that** triggers can be evaluated at runtime

**Acceptance Criteria:**
- [ ] Introduce `BoostActivation` internal type to track activation state
- [ ] Refactor admission webhook to use trigger evaluation
- [ ] Controller can evaluate triggers independently of admission
- [ ] Existing PodCreate behavior unchanged (no regression)
- [ ] Unit tests verify refactored logic

**Technical Notes:**
- Create `internal/boost/activation.go`
- Activation state includes: trigger type, start time, expiry condition
- Maintain existing webhook behavior for PodCreate trigger

---

### US-003: State Tracking Infrastructure
**As a** developer  
**I want** pod annotations to track trigger-related state  
**So that** trigger detection survives controller restarts

**Acceptance Criteria:**
- [ ] Annotation schema for last-seen restartCounts per container
- [ ] Annotation schema for last-seen condition states
- [ ] Annotation schema for last activation timestamp
- [ ] Helper functions in `internal/boost/pod/` for state management
- [ ] State can be read/written atomically

**Technical Notes:**
- Extend `internal/boost/pod/boost_annotation.go`
- Use JSON encoding for complex state
- Ensure backward compatibility with existing annotations

---

### US-004: Default Trigger Behavior
**As a** user  
**I want** existing StartupCPUBoost CRs to continue working  
**So that** I don't need to migrate configurations

**Acceptance Criteria:**
- [ ] CRs without `spec.triggers` behave exactly as before
- [ ] Default trigger (PodCreate) applied automatically
- [ ] No changes to existing pod boost behavior
- [ ] Migration guide documents new optional field

**Technical Notes:**
- Defaulting webhook or controller logic
- Verify with existing CR examples
- Update documentation

## Dependencies

- None (foundational EPIC)

## Risks

- **Medium**: Refactoring could introduce regressions
  - **Mitigation**: Comprehensive test coverage, gradual rollout
- **Low**: API schema changes require CRD updates
  - **Mitigation**: Additive changes only, backward compatible

## Success Metrics

- All existing tests pass
- No behavior changes for existing CRs
- API schema validates correctly
- Documentation updated

## Implementation Order

1. API schema changes (US-001)
2. State tracking infrastructure (US-003)
3. Internal activation model (US-002)
4. Default trigger behavior (US-004)

## Related EPICS

- EPIC-002: ContainerRestart Trigger (depends on this)
- EPIC-003: PodConditionTransition Trigger (depends on this)
- EPIC-004: Cooldown and Rate Limiting (depends on this)

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/controller`: **58.6%** overall coverage
  - `boost_controller.go`: Create/Delete handlers have **0% coverage** (critical gap)
  - `boost_pod_handler.go`: Update handler has **80% coverage**
- `internal/boost`: **93.5%** overall coverage (excellent foundation)
- `internal/boost/pod`: **80.0%** coverage (good foundation for state tracking)

### Coverage Gaps to Address

1. **Controller Event Handlers** (Critical - 0% coverage)
   - `Create()` event handler: No tests exist
   - `Delete()` event handler: No tests exist
   - **Priority**: High - Must address before implementation
   - **Estimated Tests**: 8-10 test cases

2. **Trigger Schema Validation**
   - No tests for trigger parsing/validation
   - No tests for default trigger behavior
   - **Priority**: High
   - **Estimated Tests**: 6-8 test cases

3. **State Tracking Infrastructure**
   - Existing pod annotation tests cover basic functionality
   - Need tests for trigger state storage (restartCounts, condition states, activation times)
   - **Priority**: Medium
   - **Estimated Tests**: 5-7 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] API schema validation (triggers field)
- [ ] Default trigger behavior (PodCreate when triggers omitted)
- [ ] Trigger parsing and validation
- [ ] Backward compatibility (CRs without triggers)
- [ ] Controller Create/Delete event handlers
- [ ] State tracking in pod annotations

**Coverage Targets**:
- Overall controller package: **≥80%** (currently 58.6%)
- New trigger code: **≥85%**
- Critical paths (event handlers): **≥90%**

**Test Strategy**:
- Use TDD approach for new trigger logic
- Add missing controller handler tests immediately (before implementation)
- Extend existing pod annotation tests for state tracking
- Integration tests for backward compatibility scenarios

**Estimated New Tests**: 15-20 test cases

