# Tarnished Agent - Code Review Findings

## Critical Severity

### 1. Silent Error Handling in Service Commands

**Location:** `cmd/tarnished/main.go:178-251`

`uninstall`, `start`, `stop`, `status` commands ignore config/agent initialization errors:

```go
cfg, _ := config.Load(cfgFile)  // ERROR IGNORED
ag, _ := agent.New(cfg, logger) // ERROR IGNORED
```

Will panic on nil pointer dereference if config loading fails.

**Fix:** Handle errors properly and return meaningful error messages.

---

### 2. Unsafe Type Assertion in Queue Draining

**Location:** `internal/agent/agent.go:169`

Queue stores `any` type but draining expects `*collector.Metrics`. Silent data loss if wrong type is queued.

**Fix:** Use generics or enforce type safety at queue insertion.

---

### 3. Race Condition in Service Lifecycle

**Location:** `internal/agent/agent.go:49-61`

No synchronization between `Start()` spawning goroutine and `Stop()`. If `Stop()` called before `Start()` completes initialization, `a.state` will be nil causing panic.

**Fix:** Add mutex or sync.WaitGroup to coordinate Start/Stop lifecycle.

---

## High Severity

### 5. Unchecked Type Assertion for IP Address

**Location:** `internal/registration/registration.go:93`

```go
localAddr := conn.LocalAddr().(*net.UDPAddr)
```

Panics if type assertion fails. Also uses hard-coded Google DNS (8.8.8.8) which fails in isolated networks.

**Fix:** Add type assertion check with comma-ok idiom.

---

### 6. Insecure Directory Permissions

**Location:** `internal/registration/registration.go:44`

State directory created with `0755` (world-readable):

```go
if err := os.MkdirAll(dir, 0755); err != nil {
```

Device IDs exposed to other local users.

**Fix:** Use `0700` for directory permissions.

---

### 7. No Configuration Validation

**Location:** `internal/config/config.go:32-58`

No validation for:

- URL format (could be `"abc"` instead of `"https://..."`)
- Interval bounds (could be 0 or negative)
- Config file permissions

**Fix:** Add comprehensive validation in `Validate()` function.

---

## Medium Severity

### 8. Silent Metrics Collection Failures

**Location:** `internal/collector/collector.go:54-113`

Returns valid `Metrics` struct even if all collections failed. Zero values are indistinguishable from actual data.

**Fix:** Add success flags or return partial error information with metrics.

---

### 9. Unused `Drain()` Method

**Location:** `internal/queue/queue.go:62-69`

Dead code - agent uses `Pop()` loop instead. Creates maintenance confusion.

**Fix:** Remove unused method or refactor agent to use it.

---

### 10. No HTTP Connection Configuration

**Location:** `internal/httpclient/client.go`

Missing:

- Connection pooling settings
- Read/Write timeouts (only has total timeout)
- Keep-Alive configuration
- MaxIdleConns limits

**Fix:** Configure http.Transport properly.

---

### 11. Logger Never Flushed

**Location:** `cmd/tarnished/main.go:64-85`

Zap logger created but never synced. Buffered logs lost on exit.

**Fix:** Add `defer logger.Sync()` after logger creation.

---

### 13. No Collection Overlap Protection

**Location:** `internal/agent/agent.go:110-125`

If `collectAndReport()` takes longer than the interval, ticker queues multiple events. Risk of duplicate metrics on shutdown.

**Fix:** Add mutex or skip tick if previous collection still running.

---

### 14. Includes Virtual Filesystems in Disk Metrics

**Location:** `internal/collector/collector.go:77-94`

Disk metrics include `/proc`, `/sys`, tmpfs, network mounts. Potential hangs on unavailable NFS.

**Fix:** Filter partitions by filesystem type (exclude tmpfs, proc, sysfs, nfs, etc.).

---

## Low Severity

### 16. Hostname Error Ignored

**Location:** `internal/registration/registration.go:57`

```go
hostname, _ := os.Hostname()
```

Error silently ignored. Sends empty hostname to server if retrieval fails.

**Fix:** Log warning if hostname retrieval fails.

---

### 17. Magic Numbers Throughout

**Locations:**

- Queue max size: `100` (agent.go:35)
- Max retries: `5` (agent.go:19)
- CPU sample time: `time.Second` (collector.go:58)
- HTTP timeout: `30 * time.Second` (httpclient.go:26)

**Fix:** Move to configuration or documented constants.

---

### 18. No Unit Tests

**Location:** Entire codebase

Zero `*_test.go` files. No automated verification of:

- Retry logic correctness
- Queue behavior under pressure
- Configuration precedence
- Service lifecycle

**Fix:** Add unit tests for critical paths.

---

### 19. Incomplete Error Messages

**Location:** `internal/httpclient/client.go`

Errors lack context about which step failed (DNS, connection, auth, parsing).

**Fix:** Add context to wrapped errors.

---

## Summary

| Severity | Count |
| -------- | ----- |
| Critical | 3     |
| High     | 5     |
| Medium   | 7     |
| Low      | 5     |

## Priority Order for Fixes

### Immediate (Before Production)

- [ ] Fix silent error handling in service commands (#1)
- [ ] Add Start/Stop synchronization (#3)
- [ ] Fix unchecked type assertions (#2, #6)

### Short Term

- [ ] Add config validation (#8)
- [ ] Fix directory permissions (#7)
- [ ] Add disk partition filtering (#15)
- [ ] Add unit tests (#19)
- [ ] Configure HTTP transport (#11)

### Medium Term

- [ ] Add metrics validation (#9)
- [ ] Fix backoff timing (#13)
- [ ] Add collection overlap protection (#14)
- [ ] Flush logger on exit (#12)
- [ ] Remove dead code (#10)

### Low Priority

- [ ] Add XDG config support (#15)
- [ ] Handle hostname errors (#16)
- [ ] Make magic numbers configurable (#17)
- [ ] Improve error messages (#19)
