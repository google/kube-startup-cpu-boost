# EPIC-002: ContainerRestart Trigger

## Summary

Enable CPU boost to activate when containers restart, supporting recovery scenarios where pods experience crashes, liveness probe failures, or OOM kills. This allows accelerated recovery without requiring pod recreation.

## Business Value

- Accelerates recovery from container crashes and failures
- Reduces time-to-ready for restarted containers
- Supports crash-loop recovery scenarios
- Complements existing startup boost for complete lifecycle coverage

## Goals

- Detect container restartCount increments
- Apply CPU boost via in-place resource resize
- Support container name filtering
- Prevent rapid re-boost loops (via cooldown - see EPIC-004)

## User Stories

### US-005: ContainerRestart Trigger Detection
**As a** cluster administrator  
**I want** CPU boost to activate when a container restarts  
**So that** restarted containers can recover faster

**Acceptance Criteria:**
- [ ] Controller detects `restartCount` increments in pod status
- [ ] Detection works for all containers or filtered by name
- [ ] Boost applies via in-place resource resize (not pod recreation)
- [ ] Works with existing durationPolicy for boost termination
- [ ] Handles multiple container restarts correctly

**Technical Notes:**
- Watch pod status updates in `boost_pod_handler.go`
- Compare current vs. last-seen restartCount from annotations
- Use `/resize` subresource for runtime boost application
- Store last-seen restartCounts in pod annotations

---

### US-006: Container Name Filtering
**As a** cluster administrator  
**I want** to specify which containers trigger boost on restart  
**So that** I can target specific containers in multi-container pods

**Acceptance Criteria:**
- [ ] `containerName` field optional in ContainerRestart trigger
- [ ] `containerName: "*"` matches all containers (default)
- [ ] Specific container name matches only that container
- [ ] Validation rejects invalid container names
- [ ] Documentation explains filtering behavior

**Technical Notes:**
- Add `containerName *string` to `BoostTrigger` type
- Validate container name exists in pod spec
- Filter restart detection by container name

---

### US-007: Runtime Boost Application
**As a** developer  
**I want** boost to apply at runtime without pod recreation  
**So that** restarted containers get immediate CPU boost

**Acceptance Criteria:**
- [ ] Boost applied via Kubernetes `/resize` subresource
- [ ] Works on K8s 1.27+ with InPlacePodVerticalScaling
- [ ] Original resources stored in annotations (if not already)
- [ ] Boost label added to pod (if not already)
- [ ] Error handling for unsupported clusters

**Technical Notes:**
- Extend `internal/boost/startupcpuboost.go` with runtime boost method
- Reuse resource policy calculation from webhook
- Check feature gate support before applying
- Emit events on success/failure

---

### US-008: Idempotent Boost Activation
**As a** developer  
**I want** boost activation to be idempotent  
**So that** rapid restarts don't cause duplicate boosts

**Acceptance Criteria:**
- [ ] If boost already active, skip new activation
- [ ] Track active boost state per container
- [ ] New activation allowed after boost expires
- [ ] State persisted in pod annotations

**Technical Notes:**
- Check boost annotation for active state
- Compare current resources vs. original resources
- Clear active state when boost expires

## Dependencies

- EPIC-001: Trigger System Foundation (required)
- EPIC-004: Cooldown and Rate Limiting (recommended)

## Risks

- **Medium**: Rapid restart loops could cause resource exhaustion
  - **Mitigation**: Cooldown policy (EPIC-004), idempotent activation
- **Low**: In-place resize may not be supported on all clusters
  - **Mitigation**: Feature gate check, clear error messages

## Success Metrics

- Container restarts trigger boost correctly
- Boost applies within 5 seconds of restart detection
- No duplicate boosts during active period
- Works with multi-container pods

## Implementation Order

1. ContainerRestart trigger detection (US-005)
2. Runtime boost application (US-007)
3. Container name filtering (US-006)
4. Idempotent activation (US-008)

## Related EPICS

- EPIC-001: Trigger System Foundation (prerequisite)
- EPIC-004: Cooldown and Rate Limiting (complementary)
- EPIC-005: Observability and Events (for monitoring)

## Example Configuration

```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: restart-boost
spec:
  triggers:
    - type: ContainerRestart
      containerName: "*"  # All containers
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
- `internal/controller/boost_pod_handler.go`: **80%** coverage
  - `Update()` handler: **80%** coverage (needs restart detection tests)
  - Existing tests cover condition changes, but not restartCount tracking
- `internal/boost/startupcpuboost.go`: Part of **93.5%** boost package coverage
  - Well tested for admission-time boosts
  - Missing runtime boost application tests

### Coverage Gaps to Address

1. **Container Restart Detection** (Critical - 0% coverage)
   - No tests for restartCount increment detection
   - No tests for restartCount state tracking in annotations
   - **Priority**: High
   - **Estimated Tests**: 8-10 test cases

2. **Runtime Boost Application** (Critical - 0% coverage)
   - Current tests assume admission-time boost only
   - No tests for in-place resource resize at runtime
   - No tests for `/resize` subresource usage
   - **Priority**: High
   - **Estimated Tests**: 6-8 test cases

3. **Container Name Filtering** (Medium - 0% coverage)
   - No tests for container name matching
   - No tests for wildcard vs. specific container names
   - **Priority**: Medium
   - **Estimated Tests**: 4-5 test cases

4. **Idempotent Activation** (Medium - 0% coverage)
   - No tests for preventing duplicate boosts
   - No tests for active boost state checking
   - **Priority**: Medium
   - **Estimated Tests**: 3-4 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] RestartCount increment detection in pod status
- [ ] RestartCount state tracking in pod annotations
- [ ] Container name filtering (wildcard and specific)
- [ ] Runtime boost application via `/resize` subresource
- [ ] Idempotent activation (no duplicate boosts during active period)
- [ ] Error handling for unsupported clusters (feature gate checks)
- [ ] Multi-container pod scenarios

**Coverage Targets**:
- Pod handler Update function: **≥85%** (currently 80%)
- Runtime boost application: **≥90%** (new code)
- Restart detection logic: **≥90%** (new code)

**Test Strategy**:
- Extend existing pod handler tests with restart scenarios
- Add integration tests for runtime boost application
- Test edge cases: rapid restarts, multi-container pods, feature gate failures
- Mock Kubernetes API for `/resize` subresource testing

**Estimated New Tests**: 20-25 test cases

