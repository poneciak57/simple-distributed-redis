

# Development Roadmap

## Core Principles

### Engineering Philosophy
- **Robustness over complexity**: Simple, clean solutions with solid architecture that allows iterative improvements without major refactors
- **Language choice**: Go selected for its simplicity, excellent stdlib for networking/concurrency, and focus on learning distributed systems concepts rather than language complexities
- **Test-driven**: Comprehensive test coverage to ensure correctness and robustness
- **Avoid over-engineering**: Start simple, add complexity only when needed

### Success Criteria
- Redis-cli compatible
- Fault-tolerant and distributed
- Clean, maintainable codebase
- Well-documented architecture and decisions

---

## Phase 1: Foundation (CURRENT)
**Goal**: Establish core building blocks for the distributed system

### 1.1 Protocol Layer ✅ (COMPLETED)
- [x] RESP2 protocol parser
- [x] RESP2 protocol renderer
- [x] Unit tests for parser/renderer
- [x] Support for all RESP2 data types (SimpleString, BulkString, Integer, Error, Array)

**Status**: Parser implementation complete with comprehensive tests

### 1.2 Storage Engine (IN PROGRESS)
**Deliverables**:
- [x] Storage interface definition (pluggable backend)
- [x] In-memory storage implementation
- [x] Write-Ahead Log (WAL) for durability
- [x] Snapshot mechanism for faster recovery

**Key Design Decisions**:
- Generic storage interface `Storage[T any]` for flexibility
- NOT thread-safe by design (handled at higher level)
- Simple map-based implementation to start

**Tests Required**:
- Unit tests for all storage operations
- WAL correctness tests (append, replay, truncate)
- Snapshot creation and restoration tests

---

## Phase 2: Single-Node Server
**Goal**: Working single-node Redis-compatible server

### 2.1 TCP Server
**Deliverables**:
- [x] TCP listener on configurable port
- [x] Connection handling (accept, read, write)
- [x] RESP2 protocol integration
- [x] Graceful shutdown
- [x] Connection pooling and limits
- [x] Timeout handling

**Tests Required**:
- Integration tests with redis-cli
- Concurrent connection tests
- Load testing
- Error handling tests

### 2.2 Command Router
**Deliverables**:
- [x] Route RESP2 commands to handlers
- [x] Response serialization
- [x] Error response handling
- [x] Command pipelining support

### 2.3 Observability
**Deliverables**:
- [x] Structured logging (with levels)
- [ ] Basic metrics (requests/sec, errors, latency)
- [ ] Health check endpoint (HTTP)
- [x] Configuration management

---

## Phase 3: Consensus Layer (Raft)
**Goal**: Enable distributed consensus among nodes

### 3.1 Raft Fundamentals
**Deliverables**:
- [ ] Raft state machine (Follower, Candidate, Leader)
- [ ] Leader election
- [ ] Term management
- [ ] Election timeouts and heartbeats

**Tests Required**:
- Leader election tests
- Split-brain prevention tests
- Term transition tests

### 3.2 Log Replication
**Deliverables**:
- [ ] AppendEntries RPC
- [ ] Log consistency checks
- [ ] Commit index management
- [ ] Apply committed entries to state machine

**Tests Required**:
- Log replication correctness
- Network partition handling
- Log conflict resolution

### 3.3 Cluster Membership
**Deliverables**:
- [ ] Static cluster configuration
- [ ] Node discovery
- [ ] Join/leave operations (future)

### 3.4 Integration with Storage
**Deliverables**:
- [ ] Replicated state machine
- [ ] Write operations through Raft
- [ ] Read consistency guarantees
- [ ] Snapshot integration

---

## Phase 4: Distribution & Sharding
**Goal**: Scale horizontally with data partitioning

### 4.1 Consistent Hashing
**Deliverables**:
- [ ] Hash ring implementation
- [ ] Virtual nodes for better distribution
- [ ] Node addition/removal handling
- [ ] Key routing logic

**Tests Required**:
- Distribution uniformity tests
- Rebalancing tests
- Edge case handling

### 4.2 Data Replication
**Deliverables**:
- [ ] Replication factor configuration
- [ ] Primary-replica model
- [ ] Read preference (primary/replica)
- [ ] Consistency levels

### 4.3 Cross-Node Communication
**Deliverables**:
- [ ] gRPC service definitions
- [ ] Node-to-node RPC calls
- [ ] Request forwarding
- [ ] Retry and timeout logic

**gRPC Services**:
- KeyValueService (GET, SET, DEL)
- RaftService (AppendEntries, RequestVote)
- ClusterService (Join, Leave, Health)

---

## Phase 5: Advanced Features (Optional)
**Goal**: Production-ready enhancements

### 5.1 Pub/Sub
**Deliverables**:
- [ ] PUBLISH command
- [ ] SUBSCRIBE command
- [ ] UNSUBSCRIBE command
- [ ] UDP server for pub/sub
- [ ] Channel management

### 5.2 Service Discovery
**Deliverables**:
- [ ] Static configuration (initial)
- [ ] Dynamic service discovery (future)
- [ ] Health monitoring
- [ ] Automatic failover

### 5.3 Client Features
**Deliverables**:
- [ ] Connection pooling
- [ ] Automatic retry
- [ ] Load balancing
- [ ] Circuit breaker

### 5.4 Operations
**Deliverables**:
- [ ] Docker images
- [ ] Docker Compose setup
- [ ] Kubernetes manifests (optional)
- [ ] Monitoring dashboards
- [ ] Performance profiling tools

---

## Documentation Requirements

### Must Have (Before Phase 2)
- [x] Project brief
- [x] Roadmap (this document)
- [ ] Architecture documentation
- [ ] API specification
- [ ] Design decisions log

### Nice to Have
- [ ] Deployment guide
- [ ] Troubleshooting guide
- [ ] Performance tuning guide
- [ ] Contributing guidelines
