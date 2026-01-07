# EPIC-003: PodConditionTransition Trigger

## Summary

Enable CPU boost to activate when pod conditions transition (e.g., Ready False → True), supporting recovery scenarios where pods temporarily lose readiness without container restarts. This addresses the common production issue where dependency outages cause readiness flaps.

## Business Value

- Accelerates recovery from transient dependency failures (DB restarts, network issues)
- Supports readiness-based recovery without pod restarts
- Complements container restart trigger for comprehensive recovery coverage
- Addresses real-world production scenarios documented in background discussions

## Goals

- Detect pod condition status transitions
- Support configurable condition types and status transitions
- Apply boost via in-place resource resize
- Debounce rapid condition flaps

## User Stories

### US-009: PodConditionTransition Trigger Detection
**As a** cluster administrator  
**I want** CPU boost to activate when pod conditions transition  
**So that** pods recovering from dependency failures get accelerated recovery

**Acceptance Criteria:**
- [ ] Controller detects condition status changes in pod status
- [ ] Supports any pod condition type (Ready, PodScheduled, etc.)
- [ ] Matches transitions from specific `fromStatus` to `toStatus`
- [ ] Boost applies via in-place resource resize
- [ ] Works with existing durationPolicy

**Technical Notes:**
- Watch pod status updates in `boost_pod_handler.go`
- Compare current vs. last-seen condition states from annotations
- Store last-seen condition states in pod annotations
- Evaluate transition matches against trigger configuration

---

### US-010: Condition Transition Configuration
**As a** cluster administrator  
**I want** to configure which condition transitions trigger boost  
**So that** I can target specific recovery scenarios

**Acceptance Criteria:**
- [ ] `conditionType` field specifies which condition to watch
- [ ] `fromStatus` and `toStatus` specify transition direction
- [ ] Supports status values: "True", "False", "Unknown"
- [ ] Validation ensures required fields are present
- [ ] Common pattern: Ready False → True documented

**Technical Notes:**
- Add fields to `BoostTrigger` type:
  - `ConditionType *string`
  - `FromStatus *string`
  - `ToStatus *string`
- Validate enum values for status
- Require all three fields when type is PodConditionTransition

---

### US-011: Initial Condition Handling
**As a** developer  
**I want** initial Ready=True state to not trigger boost  
**So that** boost only applies on actual transitions, not initial state

**Acceptance Criteria:**
- [ ] Boost does NOT apply on initial Ready=True (unless PodCreate also configured)
- [ ] Only transitions from non-matching to matching status trigger boost
- [ ] First condition observation stored without triggering
- [ ] Documentation explains initial state behavior

**Technical Notes:**
- Check if condition state was previously observed
- Only trigger if previous state was different
- Handle pods created with Ready=True initially

---

### US-012: Condition Flap Debouncing
**As a** developer  
**I want** rapid condition flaps to not cause continuous boosting  
**So that** system remains stable during flapping conditions

**Acceptance Criteria:**
- [ ] Cooldown policy applies to condition transitions (see EPIC-004)
- [ ] Idempotent activation prevents duplicate boosts
- [ ] State tracking prevents re-trigger during active boost
- [ ] Events logged for skipped activations

**Technical Notes:**
- Reuse cooldown logic from EPIC-004
- Check active boost state before applying
- Store last transition timestamp

## Dependencies

- EPIC-001: Trigger System Foundation (required)
- EPIC-004: Cooldown and Rate Limiting (recommended)

## Risks

- **Medium**: Readiness flapping could cause resource exhaustion
  - **Mitigation**: Cooldown policy, idempotent activation, debouncing
- **Low**: Complex condition state tracking
  - **Mitigation**: Clear annotation schema, comprehensive tests

## Success Metrics

- Condition transitions trigger boost correctly
- Initial Ready=True does not trigger (unless PodCreate configured)
- Rapid flaps are debounced appropriately
- Boost applies within 5 seconds of transition detection

## Implementation Order

1. Condition transition detection (US-009)
2. Condition transition configuration (US-010)
3. Initial condition handling (US-011)
4. Condition flap debouncing (US-012)

## Related EPICS

- EPIC-001: Trigger System Foundation (prerequisite)
- EPIC-002: ContainerRestart Trigger (complementary)
- EPIC-004: Cooldown and Rate Limiting (complementary)
- EPIC-005: Observability and Events (for monitoring)

## Example Configuration

```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: readiness-recovery-boost
spec:
  triggers:
    - type: PodConditionTransition
      conditionType: Ready
      fromStatus: "False"
      toStatus: "True"
  resourcePolicy:
    containerPolicies:
      - containerName: app
        percentageIncrease:
          value: 50
  durationPolicy:
    podCondition:
      type: Ready
      status: "True"
  cooldown:
    minIntervalSeconds: 300
    maxActivationsPerHour: 3
```

## Use Case: Database Restart Recovery

**Scenario**: Application pod becomes NotReady when database restarts, then needs to reconnect when DB comes back.

**Solution**: PodConditionTransition trigger (Ready False → True) provides CPU boost during reconnection phase, accelerating recovery without pod restart.

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/controller/boost_pod_handler.go`: **80%** coverage
  - `Update()` handler: **80%** coverage
  - Existing tests check if conditions changed, but not specific transitions
  - No tests for condition state tracking
- `internal/boost/startupcpuboost.go`: Part of **93.5%** boost package coverage
  - Well tested for admission-time boosts
  - Missing condition transition detection tests

### Coverage Gaps to Address

1. **Condition Transition Detection** (Critical - 0% coverage)
   - No tests for specific from→to transition matching
   - No tests for condition state tracking in annotations
   - Existing tests only check if conditions changed (not which transition)
   - **Priority**: High
   - **Estimated Tests**: 8-10 test cases

2. **Initial Condition Handling** (High - 0% coverage)
   - No tests for preventing false positives on initial Ready=True
   - No tests for first condition observation without triggering
   - **Priority**: High
   - **Estimated Tests**: 4-5 test cases

3. **Multiple Condition Types** (Medium - 0% coverage)
   - No tests for non-Ready conditions (PodScheduled, etc.)
   - No tests for multiple condition transitions
   - **Priority**: Medium
   - **Estimated Tests**: 3-4 test cases

4. **Condition Flap Debouncing** (Medium - 0% coverage)
   - No tests for rapid condition flaps
   - No tests for cooldown integration with condition transitions
   - **Priority**: Medium (depends on EPIC-004)
   - **Estimated Tests**: 3-4 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] Condition transition detection (fromStatus → toStatus matching)
- [ ] Condition state tracking in pod annotations
- [ ] Initial condition handling (no false positives)
- [ ] Multiple condition types (Ready, PodScheduled, etc.)
- [ ] Condition flap debouncing
- [ ] Integration with cooldown policy (EPIC-004)
- [ ] Edge cases: Unknown status, missing conditions

**Coverage Targets**:
- Pod handler Update function: **≥85%** (currently 80%)
- Condition transition logic: **≥90%** (new code)
- State tracking: **≥90%** (new code)

**Test Strategy**:
- Extend existing pod handler tests with transition scenarios
- Test all condition types and status combinations
- Test initial state vs. transition scenarios
- Integration tests for rapid flap scenarios
- Mock condition state changes for unit tests

**Estimated New Tests**: 20-25 test cases

