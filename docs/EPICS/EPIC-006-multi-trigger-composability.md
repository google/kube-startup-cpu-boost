# EPIC-006: Multi-Trigger Composability

## Summary

Enable multiple triggers to be configured per StartupCPUBoost, allowing users to combine PodCreate, ContainerRestart, and PodConditionTransition triggers in a single configuration. This provides flexibility and reduces configuration complexity.

## Business Value

- Simplifies configuration for workloads needing multiple boost scenarios
- Reduces number of StartupCPUBoost CRs needed
- Enables comprehensive lifecycle coverage in single configuration
- Supports complex production scenarios with multiple failure modes

## Goals

- Support multiple triggers in `spec.triggers[]` array
- Each trigger independently evaluates and activates boost
- DurationPolicy applies per activation (not per trigger)
- Clear semantics for overlapping activations

## User Stories

### US-024: Multiple Triggers Configuration
**As a** cluster administrator  
**I want** to configure multiple triggers in a single StartupCPUBoost  
**So that** I can cover all boost scenarios without multiple CRs

**Acceptance Criteria:**
- [ ] `spec.triggers[]` accepts array of trigger configurations
- [ ] Each trigger independently evaluated
- [ ] Multiple triggers can be active simultaneously (different containers/conditions)
- [ ] Validation ensures trigger configurations are valid
- [ ] Documentation explains multi-trigger behavior

**Technical Notes:**
- Array already supported by API schema
- Each trigger evaluated independently in controller
- Store active triggers per pod/container

---

### US-025: Independent Trigger Evaluation
**As a** developer  
**I want** each trigger to be evaluated independently  
**So that** triggers don't interfere with each other

**Acceptance Criteria:**
- [ ] PodCreate trigger evaluated at admission (existing behavior)
- [ ] ContainerRestart trigger evaluated on pod status updates
- [ ] PodConditionTransition trigger evaluated on condition changes
- [ ] Each trigger can activate boost independently
- [ ] No cross-trigger dependencies

**Technical Notes:**
- Separate evaluation logic per trigger type
- Store last-seen state per trigger type
- Evaluate all triggers on relevant pod updates

---

### US-026: Per-Activation Duration Policy
**As a** developer  
**I want** durationPolicy to apply per activation, not per trigger  
**So that** each boost activation has independent expiry

**Acceptance Criteria:**
- [ ] Each activation has independent expiry timer
- [ ] DurationPolicy evaluated per activation
- [ ] Multiple activations can be active simultaneously
- [ ] Reversion happens per activation expiry
- [ ] Clear semantics for overlapping activations

**Technical Notes:**
- Track activation state per (pod, trigger, container) combination
- Each activation has start time and expiry condition
- Revert resources when all activations expire (or use max)

---

### US-027: Activation State Management
**As a** developer  
**I want** to track multiple active activations per pod  
**So that** multi-trigger scenarios work correctly

**Acceptance Criteria:**
- [ ] Store active activation state per trigger type
- [ ] Track activation start time per trigger
- [ ] Handle expiration per activation independently
- [ ] State persisted in pod annotations
- [ ] Clear state on pod deletion

**Technical Notes:**
- Extend boost annotation to include activation array
- Each activation includes: trigger type, start time, expiry condition
- Revert logic considers all active activations

---

### US-028: Overlapping Activation Semantics
**As a** cluster administrator  
**I want** clear behavior when multiple triggers activate simultaneously  
**So that** I can reason about boost behavior

**Acceptance Criteria:**
- [ ] If boost already active, new activation extends duration (or no-op)
- [ ] Behavior documented and consistent
- [ ] Events indicate which triggers activated
- [ ] Metrics track per-trigger activations

**Technical Notes:**
- Decision: If boost active, new activation extends expiry (refresh timer)
- Alternative: No-op if already active (simpler, but less flexible)
- Document chosen approach clearly

## Dependencies

- EPIC-001: Trigger System Foundation (required)
- EPIC-002: ContainerRestart Trigger (for multi-trigger scenarios)
- EPIC-003: PodConditionTransition Trigger (for multi-trigger scenarios)

## Risks

- **Medium**: Complex state management with multiple activations
  - **Mitigation**: Clear data structures, comprehensive tests, documentation
- **Low**: Confusion about overlapping activation behavior
  - **Mitigation**: Clear documentation, examples, events

## Success Metrics

- Multiple triggers work correctly in single CR
- Each trigger activates independently
- DurationPolicy applies per activation
- No state corruption with multiple activations

## Implementation Order

1. Multiple triggers configuration (US-024)
2. Independent trigger evaluation (US-025)
3. Activation state management (US-027)
4. Per-activation duration policy (US-026)
5. Overlapping activation semantics (US-028)

## Related EPICS

- EPIC-001: Trigger System Foundation (prerequisite)
- EPIC-002: ContainerRestart Trigger (component)
- EPIC-003: PodConditionTransition Trigger (component)
- EPIC-004: Cooldown and Rate Limiting (applies per trigger)
- EPIC-005: Observability and Events (tracks per trigger)

## Example Configuration

```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: comprehensive-boost
spec:
  triggers:
    - type: PodCreate                    # Boost on initial startup
    - type: ContainerRestart             # Boost on container restart
      containerName: "*"
    - type: PodConditionTransition       # Boost on readiness recovery
      conditionType: Ready
      fromStatus: "False"
      toStatus: "True"
  cooldown:
    minIntervalSeconds: 300
    maxActivationsPerHour: 5
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

## Design Decisions

1. **Per-activation expiry**: Each activation has independent expiry
   - Rationale: Allows different triggers to have different effective durations
   
2. **Activation refresh**: New activation extends existing boost
   - Rationale: Simpler than tracking multiple independent boosts per container
   - Alternative: Track multiple boosts (more complex, but more flexible)

3. **Cooldown per trigger**: Cooldown applies per (pod, boost) not per trigger
   - Rationale: Prevents total resource exhaustion regardless of trigger source

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/controller/boost_pod_handler.go`: **80%** coverage
  - Current tests handle single scenarios
  - Need tests for multiple trigger evaluation
- `internal/boost/startupcpuboost.go`: Part of **93.5%** boost package coverage
  - Well tested for single boost scenarios
  - Need tests for multiple simultaneous activations
- `internal/boost/manager.go`: **93.5%** coverage
  - Excellent foundation, need multi-trigger scenarios

### Coverage Gaps to Address

1. **Multiple Triggers Configuration** (High - 0% coverage)
   - No tests for multiple triggers in single CR
   - No tests for trigger validation with multiple triggers
   - **Priority**: High
   - **Estimated Tests**: 4-5 test cases

2. **Independent Trigger Evaluation** (High - 0% coverage)
   - No tests for each trigger evaluated independently
   - No tests for trigger isolation (one trigger doesn't affect another)
   - **Priority**: High
   - **Estimated Tests**: 6-8 test cases

3. **Per-Activation Duration Policy** (Medium - 0% coverage)
   - No tests for independent expiry per activation
   - No tests for multiple active activations
   - **Priority**: Medium
   - **Estimated Tests**: 5-6 test cases

4. **Overlapping Activation Semantics** (Medium - 0% coverage)
   - No tests for activation refresh behavior
   - No tests for multiple triggers activating simultaneously
   - No tests for expiry when multiple activations active
   - **Priority**: Medium
   - **Estimated Tests**: 4-5 test cases

5. **State Management with Multiple Triggers** (Medium - 0% coverage)
   - No tests for tracking multiple activation states
   - No tests for state persistence with multiple triggers
   - **Priority**: Medium
   - **Estimated Tests**: 3-4 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] Multiple triggers in single CR configuration
- [ ] Independent evaluation of each trigger
- [ ] Per-activation expiry (independent timers)
- [ ] Overlapping activation behavior (refresh vs. no-op)
- [ ] State tracking for multiple activations
- [ ] Integration with cooldown (applies across all triggers)
- [ ] Edge cases: all triggers fire simultaneously, rapid sequential triggers

**Coverage Targets**:
- Multi-trigger evaluation: **≥85%** (new code)
- Activation state management: **≥90%** (complex logic)
- Integration scenarios: **≥80%** (end-to-end)

**Test Strategy**:
- Test all trigger combinations (PodCreate + ContainerRestart, etc.)
- Test independent trigger evaluation with mocks
- Integration tests for real multi-trigger scenarios
- Test activation refresh vs. independent expiry semantics
- Test edge cases: all triggers at once, rapid sequences

**Estimated New Tests**: 15-20 test cases

