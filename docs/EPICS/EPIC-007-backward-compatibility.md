# EPIC-007: Backward Compatibility and Migration

## Summary

Ensure 100% backward compatibility with existing StartupCPUBoost configurations while introducing new trigger-based activation model. Provide migration path and documentation for users adopting new features.

## Business Value

- Zero-downtime upgrade path for existing deployments
- No breaking changes for current users
- Clear migration guide for adopting new features
- Maintains trust in upgrade process

## Goals

- Existing CRs work without modification
- Default behavior unchanged (PodCreate trigger)
- Clear documentation of new vs. old behavior
- Migration examples and best practices

## User Stories

### US-029: Default Trigger Behavior
**As a** existing user  
**I want** my existing StartupCPUBoost CRs to continue working  
**So that** I can upgrade without configuration changes

**Acceptance Criteria:**
- [ ] CRs without `spec.triggers` behave exactly as before
- [ ] Default trigger (PodCreate) applied automatically
- [ ] No changes to existing pod boost behavior
- [ ] All existing tests pass without modification
- [ ] No regression in functionality

**Technical Notes:**
- Defaulting logic in controller or webhook
- If `triggers` is nil/empty, treat as `[{type: PodCreate}]`
- Verify with existing CR examples from tests/docs

---

### US-030: API Schema Backward Compatibility
**As a** developer  
**I want** API schema changes to be additive only  
**So that** existing CRs remain valid

**Acceptance Criteria:**
- [ ] New fields are optional (`+optional` kubebuilder marker)
- [ ] Existing required fields remain required
- [ ] CRD validation accepts existing CR formats
- [ ] No breaking changes to existing field semantics

**Technical Notes:**
- All new fields use `omitempty` JSON tag
- Use `+optional` kubebuilder markers
- Verify CRD generation produces compatible schema

---

### US-031: Webhook Backward Compatibility
**As a** developer  
**I want** admission webhook to maintain existing behavior  
**So that** pod creation continues to work as before

**Acceptance Criteria:**
- [ ] Webhook applies boost on CREATE (existing behavior)
- [ ] Webhook behavior unchanged for CRs without triggers
- [ ] Webhook respects default PodCreate trigger
- [ ] No changes to webhook response format

**Technical Notes:**
- Webhook continues to apply boost at admission
- Trigger evaluation defaults to PodCreate
- No changes to webhook API or response

---

### US-032: Controller Backward Compatibility
**As a** developer  
**I want** controller to handle both old and new CR formats  
**So that** mixed deployments work correctly

**Acceptance Criteria:**
- [ ] Controller processes CRs without triggers correctly
- [ ] Controller processes CRs with triggers correctly
- [ ] No errors when reading old-format CRs
- [ ] Status updates work for both formats

**Technical Notes:**
- Check for nil/empty triggers before processing
- Apply default trigger logic
- Handle missing fields gracefully

---

### US-033: Migration Documentation
**As a** cluster administrator  
**I want** documentation on migrating to new trigger model  
**So that** I can adopt new features when ready

**Acceptance Criteria:**
- [ ] Migration guide explains new features
- [ ] Examples show old vs. new configuration
- [ ] Best practices for adopting triggers
- [ ] Common migration patterns documented

**Technical Notes:**
- Create `docs/migration.md`
- Include before/after examples
- Explain when to use each trigger type
- Link to design proposal

---

### US-034: Deprecation Strategy (Future)
**As a** maintainer  
**I want** clear deprecation path for any future breaking changes  
**So that** users have time to migrate

**Acceptance Criteria:**
- [ ] No immediate deprecations (all features additive)
- [ ] Future deprecation policy documented
- [ ] Deprecation warnings use Kubernetes conventions
- [ ] Timeline for removal (if any) clearly stated

**Technical Notes:**
- Document in CONTRIBUTING.md or similar
- Use Kubernetes deprecation annotation format
- Provide migration tools if needed

## Dependencies

- EPIC-001: Trigger System Foundation (must maintain compatibility)

## Risks

- **Low**: Subtle behavior differences in edge cases
  - **Mitigation**: Comprehensive test coverage, integration tests
- **Low**: User confusion about new vs. old behavior
  - **Mitigation**: Clear documentation, examples

## Success Metrics

- All existing tests pass
- Existing CRs work without modification
- No user-reported regressions
- Migration guide usage and feedback

## Implementation Order

1. Default trigger behavior (US-029)
2. API schema backward compatibility (US-030)
3. Webhook backward compatibility (US-031)
4. Controller backward compatibility (US-032)
5. Migration documentation (US-033)
6. Deprecation strategy (US-034)

## Related EPICS

- EPIC-001: Trigger System Foundation (must be compatible)
- All other EPICS (must not break compatibility)

## Testing Strategy

### Compatibility Tests
- [ ] Existing CR examples work without modification
- [ ] Pod creation with old-format CRs works
- [ ] Status updates work for old-format CRs
- [ ] Webhook behavior unchanged for old-format CRs

### Regression Tests
- [ ] All existing unit tests pass
- [ ] All existing integration tests pass
- [ ] E2E tests with existing configurations pass

## Example: Old vs. New Configuration

### Old Configuration (Still Works)
```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: old-format
spec:
  resourcePolicy:
    containerPolicies:
      - containerName: app
        percentageIncrease:
          value: 50
  durationPolicy:
    podCondition:
      type: Ready
      status: "True"
```

### New Configuration (Equivalent)
```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: new-format
spec:
  triggers:
    - type: PodCreate  # Explicit, but same as default
  resourcePolicy:
    containerPolicies:
      - containerName: app
        percentageIncrease:
          value: 50
  durationPolicy:
    podCondition:
      type: Ready
      status: "True"
```

### New Configuration (Enhanced)
```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: enhanced-format
spec:
  triggers:
    - type: PodCreate
    - type: ContainerRestart      # New capability
    - type: PodConditionTransition # New capability
      conditionType: Ready
      fromStatus: "False"
      toStatus: "True"
  cooldown:                        # New capability
    minIntervalSeconds: 300
  resourcePolicy:
    containerPolicies:
      - containerName: app
        percentageIncrease:
          value: 50
  durationPolicy:
    podCondition:
      type: Ready
      status: "True"
```

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/controller`: **58.6%** overall coverage
  - Need tests for default trigger behavior
- `internal/webhook`: **77.5%** coverage
  - Well tested, need backward compatibility verification
- `internal/boost`: **93.5%** coverage
  - Excellent foundation, need default behavior tests
- API schema: CRD generation (needs validation tests)

### Coverage Gaps to Address

1. **Default Trigger Behavior** (Critical - 0% coverage)
   - No tests for default PodCreate trigger when triggers omitted
   - No tests for backward compatibility with existing CRs
   - **Priority**: Critical - Must verify before release
   - **Estimated Tests**: 6-8 test cases

2. **API Schema Backward Compatibility** (High - 0% coverage)
   - No tests for CRD validation accepting old-format CRs
   - No tests for optional field handling
   - **Priority**: High
   - **Estimated Tests**: 4-5 test cases

3. **Webhook Backward Compatibility** (High - Partial coverage)
   - Existing webhook tests cover current behavior
   - Need explicit tests that webhook behavior unchanged for old CRs
   - **Priority**: High
   - **Estimated Tests**: 3-4 test cases

4. **Controller Backward Compatibility** (High - Partial coverage)
   - Need tests for controller handling both old and new CR formats
   - Need tests for status updates working with both formats
   - **Priority**: High
   - **Estimated Tests**: 4-5 test cases

5. **Migration Scenarios** (Medium - 0% coverage)
   - No tests for gradual migration (some CRs old, some new)
   - No tests for mixed deployments
   - **Priority**: Medium
   - **Estimated Tests**: 2-3 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] Default trigger (PodCreate) when triggers field omitted
- [ ] Existing CRs work without modification
- [ ] CRD validation accepts old-format CRs
- [ ] Webhook behavior unchanged for old CRs
- [ ] Controller handles both formats correctly
- [ ] Status updates work for both formats
- [ ] Mixed deployment scenarios (old + new CRs)
- [ ] No regression in existing functionality

**Coverage Targets**:
- Backward compatibility paths: **≥95%** (critical for safe upgrade)
- Default behavior: **≥90%** (must be bulletproof)
- Migration scenarios: **≥80%** (important but less critical)

**Test Strategy**:
- Comprehensive regression tests with existing CR examples
- Test default trigger behavior explicitly
- Integration tests with real old-format CRs
- Test mixed scenarios (old and new CRs in same cluster)
- Verify all existing tests still pass (no regressions)
- Test CRD validation with both formats

**Estimated New Tests**: 15-20 test cases

### Regression Test Requirements

**Critical**: All existing tests must continue to pass
- [ ] All existing unit tests pass
- [ ] All existing integration tests pass
- [ ] All existing E2E tests pass
- [ ] No behavior changes for existing CRs
- [ ] Performance characteristics unchanged

