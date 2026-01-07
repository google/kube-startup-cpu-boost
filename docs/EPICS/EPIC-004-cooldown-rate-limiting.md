# EPIC-004: Cooldown and Rate Limiting

## Summary

Implement cooldown and rate limiting controls to prevent pathological re-trigger loops during crash loops, readiness flapping, or other rapid failure scenarios. This ensures system stability and prevents resource exhaustion.

## Business Value

- Prevents resource exhaustion from rapid boost activations
- Protects cluster stability during failure scenarios
- Provides predictable behavior during crash loops
- Enables safe use of lifecycle triggers in production

## Goals

- Implement minimum interval between activations
- Implement maximum activations per time window
- Store cooldown state in pod annotations
- Emit events for skipped activations

## User Stories

### US-013: Cooldown Policy API
**As a** cluster administrator  
**I want** to configure cooldown limits for boost activations  
**So that** I can prevent excessive boosting during failure scenarios

**Acceptance Criteria:**
- [ ] `spec.cooldown` field added to `StartupCPUBoostSpec`
- [ ] `minIntervalSeconds` specifies minimum time between activations
- [ ] `maxActivationsPerHour` specifies rate limit
- [ ] Both fields optional (cooldown is opt-in)
- [ ] Validation ensures positive values

**Technical Notes:**
- Add `CooldownPolicy` type to API schema
- Use kubebuilder validation markers
- Document default behavior (no cooldown if omitted)

---

### US-014: Minimum Interval Enforcement
**As a** developer  
**I want** minimum interval between activations to be enforced  
**So that** rapid triggers don't cause continuous boosting

**Acceptance Criteria:**
- [ ] Check time since last activation before applying boost
- [ ] Skip activation if interval not met
- [ ] Store last activation timestamp in pod annotations
- [ ] Timestamp survives controller restarts
- [ ] Works per (pod, boost) combination

**Technical Notes:**
- Store `lastActivationTime` in pod annotation
- Compare with current time before activation
- Use RFC3339 timestamp format
- Handle missing timestamp (first activation)

---

### US-015: Rate Limiting (Max Activations Per Hour)
**As a** developer  
**I want** to limit number of activations per hour  
**So that** crash loops don't exhaust resources

**Acceptance Criteria:**
- [ ] Track activation count within rolling hour window
- [ ] Skip activation if limit exceeded
- [ ] Store activation history in pod annotations
- [ ] Clean up old activation timestamps (>1 hour)
- [ ] Works per (pod, boost) combination

**Technical Notes:**
- Store array of activation timestamps in annotation
- Filter timestamps older than 1 hour
- Count remaining timestamps
- Limit annotation size (max ~100 timestamps)

---

### US-016: Cooldown Event Emission
**As a** cluster administrator  
**I want** to see when activations are skipped due to cooldown  
**So that** I can monitor and tune cooldown policies

**Acceptance Criteria:**
- [ ] Emit Kubernetes Event when activation skipped
- [ ] Event includes reason (minInterval or maxActivations)
- [ ] Event includes relevant timestamps
- [ ] Events are searchable and observable

**Technical Notes:**
- Use controller-runtime event recorder
- Event type: `Warning`
- Event reason: `BoostSkippedCooldown`
- Include pod and boost names in event

---

### US-017: Cooldown State Persistence
**As a** developer  
**I want** cooldown state to survive controller restarts  
**So that** cooldown enforcement is reliable

**Acceptance Criteria:**
- [ ] Cooldown state stored in pod annotations (not controller memory)
- [ ] State can be read after controller restart
- [ ] State format is versioned for future changes
- [ ] Backward compatible with pods without cooldown state

**Technical Notes:**
- Use pod annotations (not controller cache)
- JSON encoding for complex state
- Include version field in state structure
- Handle missing annotations gracefully

## Dependencies

- EPIC-001: Trigger System Foundation (required)
- EPIC-002: ContainerRestart Trigger (benefits from this)
- EPIC-003: PodConditionTransition Trigger (benefits from this)

## Risks

- **Low**: Annotation size limits with many activations
  - **Mitigation**: Limit stored timestamps, cleanup old entries
- **Low**: Clock skew in multi-controller deployments
  - **Mitigation**: Use pod timestamps where possible, document limitation

## Success Metrics

- Cooldown prevents rapid re-activations
- Events emitted for all skipped activations
- State persists across controller restarts
- No performance degradation from cooldown checks

## Implementation Order

1. Cooldown policy API (US-013)
2. Minimum interval enforcement (US-014)
3. Rate limiting (US-015)
4. Cooldown state persistence (US-017)
5. Cooldown event emission (US-016)

## Related EPICS

- EPIC-001: Trigger System Foundation (prerequisite)
- EPIC-002: ContainerRestart Trigger (benefits from this)
- EPIC-003: PodConditionTransition Trigger (benefits from this)
- EPIC-005: Observability and Events (complementary)

## Example Configuration

```yaml
apiVersion: autoscaling.x-k8s.io/v1alpha1
kind: StartupCPUBoost
metadata:
  name: cooldown-example
spec:
  triggers:
    - type: ContainerRestart
    - type: PodConditionTransition
      conditionType: Ready
      fromStatus: "False"
      toStatus: "True"
  cooldown:
    minIntervalSeconds: 300      # 5 minutes between activations
    maxActivationsPerHour: 3    # Max 3 activations per hour
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

1. **Per-pod cooldown**: Each pod has independent cooldown state
   - Rationale: Different pods may have different failure patterns
   
2. **Annotation-based storage**: Cooldown state in pod annotations
   - Rationale: Survives controller restarts, visible for debugging
   
3. **Opt-in cooldown**: Cooldown is optional
   - Rationale: Backward compatibility, allows per-use-case tuning

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/boost/startupcpuboost.go`: Part of **93.5%** boost package coverage
  - Well tested for existing functionality
  - **No cooldown logic exists** - all new code
- `internal/controller/boost_pod_handler.go`: **80%** coverage
  - Will integrate cooldown checks into trigger evaluation

### Coverage Gaps to Address

1. **Cooldown Policy API** (Critical - 0% coverage)
   - No cooldown logic exists currently
   - Need tests for API schema validation
   - **Priority**: High
   - **Estimated Tests**: 3-4 test cases

2. **Minimum Interval Enforcement** (Critical - 0% coverage)
   - No tests for time-based cooldown checks
   - No tests for last activation timestamp tracking
   - **Priority**: High
   - **Estimated Tests**: 6-8 test cases

3. **Rate Limiting** (High - 0% coverage)
   - No tests for activation count tracking
   - No tests for rolling hour window calculation
   - No tests for annotation cleanup (old timestamps)
   - **Priority**: High
   - **Estimated Tests**: 5-7 test cases

4. **Cooldown State Persistence** (Medium - 0% coverage)
   - No tests for annotation-based state storage
   - No tests for state recovery after controller restart
   - No tests for state corruption handling
   - **Priority**: Medium
   - **Estimated Tests**: 3-4 test cases

5. **Event Emission** (Medium - 0% coverage)
   - No tests for skipped activation events
   - No tests for event content (reason, timestamps)
   - **Priority**: Medium (depends on EPIC-005)
   - **Estimated Tests**: 2-3 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] Cooldown policy validation (minIntervalSeconds, maxActivationsPerHour)
- [ ] Minimum interval enforcement (time since last activation)
- [ ] Maximum activations per hour (rolling window calculation)
- [ ] Cooldown state persistence in pod annotations
- [ ] State recovery after controller restart
- [ ] Edge cases: clock skew, annotation corruption, missing state
- [ ] Event emission for skipped activations
- [ ] Integration with all trigger types (ContainerRestart, PodConditionTransition)

**Coverage Targets**:
- Cooldown logic: **≥90%** (all new code, critical safety feature)
- State persistence: **≥85%** (new code)
- Integration points: **≥85%** (extends existing code)

**Test Strategy**:
- Comprehensive unit tests for cooldown logic (isolated)
- Integration tests with actual trigger evaluation
- Test edge cases: rapid triggers, state corruption, clock issues
- Mock time for deterministic testing
- Test with all trigger types to ensure cooldown applies universally

**Estimated New Tests**: 15-20 test cases

