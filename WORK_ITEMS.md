# Work Items - Kilometers CLI

## 🚧 Incomplete Features & TODOs

Last Updated: 2024-08-14

### 1. 🔐 **Digital Signature Validation** [HIGH PRIORITY]
**Location**: `internal/plugins/discovery.go:195-199`
**Status**: Stub implementation only

#### Current State:
- ✅ Interface defined (`ValidateSignature`)
- ✅ Called during plugin loading
- ❌ No actual cryptographic verification
- ❌ Signature files are just placeholders

#### Work Required:
- [ ] Implement RSA/ECDSA signature verification
- [ ] Create public/private key infrastructure
- [ ] Add binary hash verification
- [ ] Implement certificate chain validation
- [ ] Update build script to generate real signatures
- [ ] Add key rotation mechanism

#### Files to Modify:
- `internal/plugins/discovery.go` - Implement real validation
- `build-simple-plugin.sh` - Generate real signatures
- New file: `internal/plugins/crypto.go` - Crypto utilities

---

### 2. 📦 **Plugin Manifest System** [MEDIUM PRIORITY]
**Location**: `internal/plugins/discovery.go:204-225`
**Status**: Basic implementation, missing validation

#### Current State:
- ✅ Manifest files created during build
- ✅ Basic JSON structure
- ❌ No schema validation
- ❌ Missing required fields enforcement

#### Work Required:
- [ ] Add manifest schema validation
- [ ] Implement version compatibility checks
- [ ] Add dependency resolution
- [ ] Include minimum/maximum km version
- [ ] Add plugin capabilities declaration

---

### 3. 🔄 **Message Routing to Plugins** [COMPLETED - NEEDS TESTING]
**Location**: `internal/application/services/monitor_service.go`
**Status**: Implemented but needs integration testing

#### Current State:
- ✅ Plugin receives messages via HandleMessage
- ✅ File logging works
- ⚠️ Limited production testing
- ❌ No error recovery mechanism

#### Work Required:
- [ ] Add comprehensive integration tests
- [ ] Implement retry logic for failed messages
- [ ] Add message buffering for high throughput
- [ ] Create performance benchmarks
- [ ] Add plugin health checks

---

### 4. 🛡️ **Request Filtering - Performance Comparison** [NEW - HIGH PRIORITY]
**Location**: New implementation needed
**Status**: Architecture design phase

#### Objective:
Implement MCP request filtering with blacklist capability in TWO ways to benchmark performance differences:
1. **Premium Plugin Approach** - Local processing via plugin
2. **API Approach** - Remote processing via POST endpoint

#### Implementation A: Premium Plugin (Local)
- [ ] Create new premium-tier plugin `km-plugin-request-filter`
- [ ] Implement blacklist configuration loading
- [ ] Add request interception in `HandleMessage`
- [ ] Block requests matching blacklist patterns
- [ ] Return error response for blocked requests
- [ ] Add metrics collection (latency, throughput)

#### Implementation B: API POST (Remote)
- [ ] Create new API endpoint `/api/v1/filter/evaluate`
- [ ] Send every MCP request to API for evaluation
- [ ] API checks against centralized blacklist
- [ ] Return allow/block decision
- [ ] Handle API response and block if needed
- [ ] Add metrics collection (latency, network overhead)

#### Benchmark Metrics to Collect:
- [ ] **Latency**: Time from request arrival to decision
- [ ] **Throughput**: Requests per second handled
- [ ] **CPU Usage**: Local processing overhead
- [ ] **Memory Usage**: Plugin vs API client memory
- [ ] **Network Overhead**: Bandwidth for API approach
- [ ] **Failure Rate**: Dropped requests under load
- [ ] **P50/P95/P99**: Latency percentiles

#### Test Scenarios:
- [ ] Single request latency comparison
- [ ] High throughput (1000 req/s)
- [ ] Large blacklist (10,000 patterns)
- [ ] Network failure resilience (API approach)
- [ ] Cold start performance
- [ ] Concurrent request handling

#### Expected Outcomes:
- **Plugin Approach**: Lower latency, higher throughput, no network dependency
- **API Approach**: Centralized control, easier updates, network overhead
- **Trade-offs Documentation**: Guide for choosing approach

#### Files to Create:
- `internal/plugins/request-filter/` - Premium plugin implementation
- `internal/api/filter/` - API client for filtering
- `test/benchmarks/filter_comparison.go` - Benchmark suite
- `docs/FILTER_PERFORMANCE.md` - Results documentation

---

### 5. 🏢 **Enterprise Features** [NOT STARTED]
**Location**: Multiple files
**Status**: Planned but not implemented

#### Work Required:
- [ ] Custom plugin loading from external sources
- [ ] Plugin marketplace integration
- [ ] Advanced analytics plugins
- [ ] Message transformation plugins
- [ ] Rate limiting and quotas
- [ ] Multi-tenant plugin isolation

---

### 5. 🔑 **Authentication Cache** [PARTIAL]
**Location**: `internal/plugins/auth_cache.go`
**Status**: Basic in-memory cache

#### Current State:
- ✅ 5-minute TTL cache
- ✅ In-memory storage
- ❌ No persistence
- ❌ No distributed cache support

#### Work Required:
- [ ] Add Redis cache option
- [ ] Implement cache warming
- [ ] Add cache metrics
- [ ] Handle cache invalidation on tier changes
- [ ] Add cache encryption for sensitive data

---

### 6. 🚨 **Error Handling & Recovery** [NEEDS IMPROVEMENT]
**Location**: Throughout codebase
**Status**: Basic error handling only

#### Work Required:
- [ ] Add circuit breaker for plugin failures
- [ ] Implement graceful degradation
- [ ] Add detailed error telemetry
- [ ] Create error recovery strategies
- [ ] Add plugin crash detection and restart

---

### 7. 📊 **Monitoring & Observability** [NOT STARTED]
**Location**: New functionality needed
**Status**: No monitoring infrastructure

#### Work Required:
- [ ] Add OpenTelemetry integration
- [ ] Create Prometheus metrics
- [ ] Add distributed tracing
- [ ] Implement health check endpoints
- [ ] Add performance profiling

---

### 8. 📝 **API Integration** [PARTIAL]
**Location**: `internal/infrastructure/http/`
**Status**: Basic client exists

#### Current State:
- ✅ Basic HTTP client
- ✅ Authentication endpoint
- ❌ No retry logic
- ❌ No connection pooling

#### Work Required:
- [ ] Add exponential backoff retry
- [ ] Implement connection pooling
- [ ] Add request/response logging
- [ ] Create mock server for testing
- [ ] Add API versioning support

---

### 9. 🧪 **Testing Infrastructure** [NEEDS EXPANSION]
**Location**: `test/` directory
**Status**: Minimal test coverage

#### Work Required:
- [ ] Add unit tests for all components
- [ ] Create integration test suite
- [ ] Add end-to-end tests
- [ ] Implement load testing
- [ ] Add fuzzing tests for security
- [ ] Create CI/CD pipeline

---

### 10. 🔍 **Plugin Discovery** [ENHANCEMENT NEEDED]
**Location**: `internal/plugins/discovery.go`
**Status**: Basic filesystem scanning

#### Current State:
- ✅ Scans predefined directories
- ❌ No hot reload
- ❌ No remote plugin discovery

#### Work Required:
- [ ] Add filesystem watcher for hot reload
- [ ] Implement plugin registry client
- [ ] Add plugin version management
- [ ] Create plugin dependency resolution
- [ ] Add parallel plugin loading

---

## 📋 Quick Wins (Can be done in <2 hours)

1. **Add retry logic to API calls** - `internal/infrastructure/http/client.go`
2. **Implement basic health check** - New endpoint in monitor service
3. **Add plugin loading metrics** - Count success/failure in manager
4. **Create plugin test harness** - Standalone test runner
5. **Add configuration validation** - Validate all config on startup

---

## 🎯 Priority Matrix

| Priority | Feature | Effort | Impact |
|----------|---------|--------|--------|
| P0 | Request Filter Comparison | High | Critical |
| P0 | Digital Signatures | High | Critical |
| P0 | Error Recovery | Medium | High |
| P1 | Testing Infrastructure | High | High |
| P1 | API Retry Logic | Low | Medium |
| P2 | Monitoring | Medium | Medium |
| P2 | Plugin Hot Reload | Medium | Low |
| P3 | Enterprise Features | High | Low |

---

## 📝 Notes for Demo

- **Working Today**: Basic plugin system with file logging
- **Security Gap**: Signature validation is stubbed
- **Next Sprint**: Focus on P0 items (Request Filter Comparison)
- **Technical Debt**: Testing coverage is minimal
- **Quick Win**: Can add retry logic during demo if needed

### Request Filter Performance Comparison - Implementation Timeline
- **Day 1-2**: Build premium plugin with blacklist filtering
- **Day 2-3**: Create API endpoint and client integration
- **Day 4**: Set up benchmark harness and metrics collection
- **Day 5**: Run performance tests and document results
- **Expected Finding**: Plugin approach ~10-100x faster (no network hop)