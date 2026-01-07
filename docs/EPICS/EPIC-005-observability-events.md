# EPIC-005: Observability and Events

## Summary

Enhance observability for lifecycle-aware CPU boosting through Kubernetes Events and Prometheus metrics. This enables operators to understand when and why boosts are applied, skipped, or expired.

## Business Value

- Enables debugging of boost behavior in production
- Provides audit trail for boost activations
- Supports monitoring and alerting on boost patterns
- Helps tune trigger and cooldown configurations

## Goals

- Emit Kubernetes Events for boost lifecycle events
- Add Prometheus metrics for boost activations
- Include trigger type in observability data
- Support troubleshooting and performance analysis

## User Stories

### US-018: Boost Activation Events
**As a** cluster administrator  
**I want** to see Kubernetes Events when boosts are activated  
**So that** I can monitor boost behavior and troubleshoot issues

**Acceptance Criteria:**
- [ ] Event emitted when boost activates via any trigger
- [ ] Event includes trigger type (PodCreate, ContainerRestart, PodConditionTransition)
- [ ] Event includes pod and boost names
- [ ] Event type: `Normal`
- [ ] Events are searchable via `kubectl get events`

**Technical Notes:**
- Use controller-runtime event recorder
- Event reason: `BoostActivated`
- Include relevant metadata (trigger type, container name if applicable)

---

### US-019: Boost Skipped Events
**As a** cluster administrator  
**I want** to see Events when boosts are skipped  
**So that** I can understand why boosts didn't activate

**Acceptance Criteria:**
- [ ] Event emitted when boost skipped due to cooldown
- [ ] Event emitted when boost skipped due to active boost (idempotency)
- [ ] Event includes reason for skipping
- [ ] Event type: `Warning` for cooldown, `Normal` for idempotency
- [ ] Events include relevant timestamps

**Technical Notes:**
- Event reasons: `BoostSkippedCooldown`, `BoostSkippedActive`
- Include cooldown details (time remaining, activation count)
- Distinguish between different skip reasons

---

### US-020: Boost Expiration Events
**As a** cluster administrator  
**I want** to see Events when boosts expire  
**So that** I can track boost lifecycle end-to-end

**Acceptance Criteria:**
- [ ] Event emitted when boost expires (durationPolicy met)
- [ ] Event includes duration policy type
- [ ] Event includes total boost duration
- [ ] Event type: `Normal`

**Technical Notes:**
- Event reason: `BoostExpired`
- Calculate duration from activation time
- Include policy details (fixed duration or condition)

---

### US-021: Boost Activation Metrics
**As a** site reliability engineer  
**I want** Prometheus metrics for boost activations  
**So that** I can monitor boost patterns and create alerts

**Acceptance Criteria:**
- [ ] Counter metric: `cpu_boost_activations_total{trigger, boost, namespace}`
- [ ] Gauge metric: `cpu_boost_active{boost, namespace}` (number of active boosts)
- [ ] Counter metric: `cpu_boost_skipped_total{reason, boost, namespace}`
- [ ] Metrics follow Prometheus best practices
- [ ] Metrics documented

**Technical Notes:**
- Extend `internal/metrics/metrics.go`
- Use existing metrics infrastructure
- Add trigger type label to activation counter
- Update metrics on activation, expiration, skip

---

### US-022: Boost Duration Metrics
**As a** site reliability engineer  
**I want** metrics for boost duration  
**So that** I can analyze boost effectiveness

**Acceptance Criteria:**
- [ ] Histogram metric: `cpu_boost_duration_seconds{trigger, boost, namespace}`
- [ ] Tracks time from activation to expiration
- [ ] Configurable buckets for common durations
- [ ] Metrics exposed at `/metrics` endpoint

**Technical Notes:**
- Use Prometheus histogram type
- Record duration when boost expires
- Include trigger type in labels

---

### US-023: Status Field Enhancements
**As a** cluster administrator  
**I want** boost status to include activation information  
**So that** I can see boost state via `kubectl get startupcpuboost`

**Acceptance Criteria:**
- [ ] `status.lastActivationTime` field (optional)
- [ ] `status.lastTriggerType` field (optional)
- [ ] `status.skippedActivationsTotal` field (optional)
- [ ] Status updated on activation and skip events

**Technical Notes:**
- Extend `StartupCPUBoostStatus` in API schema
- Update status in controller reconcile loop
- Include in existing status update logic

## Dependencies

- EPIC-001: Trigger System Foundation (required)
- EPIC-002: ContainerRestart Trigger (for trigger-specific events)
- EPIC-003: PodConditionTransition Trigger (for trigger-specific events)
- EPIC-004: Cooldown and Rate Limiting (for skip events)

## Risks

- **Low**: Event volume could be high in large clusters
  - **Mitigation**: Events are rate-limited by Kubernetes, use metrics for high-volume data
- **Low**: Metrics cardinality with many boosts
  - **Mitigation**: Limit label combinations, use aggregation

## Success Metrics

- All boost lifecycle events emit Events
- Metrics accurately reflect boost behavior
- Events searchable and useful for debugging
- No performance impact from observability

## Implementation Order

1. Boost activation events (US-018)
2. Boost activation metrics (US-021)
3. Boost skipped events (US-019)
4. Boost expiration events (US-020)
5. Boost duration metrics (US-022)
6. Status field enhancements (US-023)

## Related EPICS

- EPIC-001: Trigger System Foundation (prerequisite)
- EPIC-002: ContainerRestart Trigger (triggers events)
- EPIC-003: PodConditionTransition Trigger (triggers events)
- EPIC-004: Cooldown and Rate Limiting (triggers skip events)

## Example Event Output

```bash
$ kubectl get events --field-selector involvedObject.name=my-pod

LAST SEEN   TYPE     REASON                OBJECT           MESSAGE
2m          Normal   BoostActivated        pod/my-pod       CPU boost activated via ContainerRestart trigger
1m          Warning  BoostSkippedCooldown  pod/my-pod       CPU boost skipped: minInterval not met (120s remaining)
30s         Normal   BoostExpired          pod/my-pod       CPU boost expired after 90s (PodCondition policy)
```

## Example Metrics

```prometheus
# Total boost activations by trigger type
cpu_boost_activations_total{trigger="ContainerRestart",boost="restart-boost",namespace="demo"} 15
cpu_boost_activations_total{trigger="PodConditionTransition",boost="readiness-boost",namespace="demo"} 8

# Currently active boosts
cpu_boost_active{boost="restart-boost",namespace="demo"} 3

# Skipped activations
cpu_boost_skipped_total{reason="cooldown",boost="restart-boost",namespace="demo"} 5
```

## Test Coverage Analysis

### Current Coverage Status

**Affected Packages**:
- `internal/metrics`: Part of metrics package (coverage not measured separately)
  - Existing metrics tests cover basic functionality
  - Need tests for new trigger-specific metrics
- `internal/controller/boost_controller.go`: **58.6%** overall
  - Status update logic has **77.8%** coverage
  - Need tests for new status fields
- Event emission: Uses controller-runtime event recorder (needs integration tests)

### Coverage Gaps to Address

1. **Boost Activation Events** (High - 0% coverage)
   - No tests for event emission on activation
   - No tests for event content (trigger type, pod/boost names)
   - **Priority**: High
   - **Estimated Tests**: 4-5 test cases

2. **Boost Skipped Events** (High - 0% coverage)
   - No tests for event emission on cooldown skip
   - No tests for event emission on idempotency skip
   - No tests for event reason/content
   - **Priority**: High
   - **Estimated Tests**: 3-4 test cases

3. **Boost Expiration Events** (Medium - 0% coverage)
   - No tests for event emission on expiration
   - No tests for duration calculation in events
   - **Priority**: Medium
   - **Estimated Tests**: 2-3 test cases

4. **Prometheus Metrics** (High - Partial coverage)
   - Existing metrics tests cover basic functionality
   - Need tests for trigger-specific labels
   - Need tests for new metrics (activations_total, skipped_total, duration histogram)
   - **Priority**: High
   - **Estimated Tests**: 5-6 test cases

5. **Status Field Updates** (Medium - Partial coverage)
   - Existing status tests cover basic fields
   - Need tests for new fields (lastActivationTime, lastTriggerType, skippedActivationsTotal)
   - **Priority**: Medium
   - **Estimated Tests**: 3-4 test cases

### Test Requirements

**Required Coverage Areas**:
- [ ] Event emission for boost activations (all trigger types)
- [ ] Event emission for skipped activations (cooldown, idempotency)
- [ ] Event emission for boost expiration
- [ ] Prometheus metrics with trigger labels
- [ ] New metrics: activations_total, skipped_total, duration_seconds
- [ ] Status field updates (lastActivationTime, lastTriggerType, etc.)
- [ ] Event content validation (reason, message, involved object)

**Coverage Targets**:
- Event emission: **≥85%** (new code)
- Metrics updates: **≥90%** (extends existing, critical for observability)
- Status updates: **≥85%** (extends existing)

**Test Strategy**:
- Mock event recorder for unit tests
- Integration tests with real event emission
- Test metrics with all trigger types
- Verify status updates in controller reconcile loop
- Test edge cases: missing events, metric label combinations

**Estimated New Tests**: 10-15 test cases

