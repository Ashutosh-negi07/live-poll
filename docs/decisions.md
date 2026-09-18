# Architectural & Engineering Decisions Log (`docs/decisions.md`)

This document tracks every major architectural, system design, database, concurrency, and performance decision made in this project. Each decision is structured like an **in-depth technical interview questionnaire**, detailing:
1. **The Core Decision**
2. **Why We Chose It (The Technical Justification)**
3. **Alternatives Considered & Why They Were Rejected**
4. **Trade-offs & Edge Cases**

---

## 📋 Table of Decisions

1. [Real-Time Protocol: Server-Sent Events (SSE) vs WebSockets vs Polling](#decision-1-real-time-protocol-server-sent-events-sse-vs-websockets-vs-polling)
2. [High-Concurrency Vote Counting: Redis `HINCRBY` vs Database Increments](#decision-2-high-concurrency-vote-counting-redis-hincrby-vs-database-increments)
3. [Duplicate Vote Prevention: Redis Sets (`SADD`) vs Database Lookups vs Client Cookies](#decision-3-duplicate-vote-prevention-redis-sets-sadd-vs-database-lookups-vs-client-cookies)
4. [Storage Strategy: Dual-Store (Redis + MongoDB) vs Single Database](#decision-4-storage-strategy-dual-store-redis--mongodb-vs-single-database)
5. [Voter Identity: Salted SHA-256 IP Hashing vs Mandatory Login vs Raw IPs](#decision-5-voter-identity-salted-sha-256-ip-hashing-vs-mandatory-login-vs-raw-ips)
6. [Backend Technology: Go (Gin) vs Node.js (Express) vs Python (FastAPI)](#decision-6-backend-technology-go-gin-vs-nodejs-express-vs-python-fastapi)
7. [Authentication Architecture: Stateless JWT vs Stateful Server Sessions](#decision-7-authentication-architecture-stateless-jwt-vs-stateful-server-sessions)
8. [Durability Strategy: Async MongoDB Persistence vs Synchronous Two-Phase Writes](#decision-8-durability-strategy-async-mongodb-persistence-vs-synchronous-two-phase-writes)
9. [Frontend Framework: React (Vite SPA) vs Next.js (SSR) vs Vanilla JS](#decision-9-frontend-framework-react-vite-spa-vs-nextjs-ssr-vs-vanilla-js)
10. [Local Infrastructure: Docker Compose vs Native Host Installations](#decision-10-local-infrastructure-docker-compose-vs-native-host-installations)

---

### Decision 1: Real-Time Protocol: Server-Sent Events (SSE) vs WebSockets vs Polling

#### ❓ Interview Question:
> *"Why did you use Server-Sent Events (SSE) instead of WebSockets for streaming live poll results? Isn't WebSocket the industry standard for real-time applications?"*

#### ✅ Decision Taken:
We chose **Server-Sent Events (SSE)** over standard HTTP/1.1 or HTTP/2 for pushing real-time vote updates to connected audience screens, while keeping vote submission as a standard HTTP `POST` request.

#### 🎯 Why We Chose It:
1. **Asymmetric Traffic Pattern:** Live polling is fundamentally unidirectional. An audience member casts a vote once or twice (a discrete HTTP action), but continuously listens to incoming aggregate results. Full-duplex communication (WebSockets) is unnecessary overhead.
2. **Native Auto-Reconnection:** The browser’s native JavaScript `EventSource` API automatically reconnects if mobile connectivity flickers, without requiring custom client-side heartbeat and retry libraries.
3. **HTTP Infrastructure Friendly:** SSE runs over standard HTTP (`Content-Type: text/event-stream`). It effortlessly passes through corporate firewalls, API gateways, load balancers, and Cloudflare CDNs without requiring protocol upgrade negotiations (`101 Switching Protocols`).
4. **Lightweight on the Server:** In Go, an SSE connection is just an open HTTP handler waiting on a Go channel and flushing text frames via `http.Flusher`.

#### ❌ Alternatives Rejected:
- **WebSockets:** Rejected because WebSockets require managing custom connection state, heartbeats/pings, protocol upgrade handshakes, and special proxy configurations. It introduces architectural complexity without offering any benefit for a one-way result stream.
- **Short Polling (`setInterval` every 1s):** Rejected because hundreds of clients hammering `GET /api/polls/:id` every second creates massive database CPU spikes, bandwidth waste, and is not truly real-time.
- **Long Polling:** Rejected because holding hanging HTTP requests open and repeatedly tearing down and rebuilding TCP connections incurs unnecessary latency and resource churn compared to a single persistent SSE stream.

---

### Decision 2: High-Concurrency Vote Counting: Redis `HINCRBY` vs Database Increments

#### ❓ Interview Question:
> *"How do you handle hundreds of votes submitted at the exact same millisecond during a live event without losing votes or causing race conditions?"*

#### ✅ Decision Taken:
We delegated the active vote counters entirely to **Redis In-Memory Hashes using the atomic `HINCRBY` command** (`HINCRBY poll:<id>:votes <option_id> 1`).

#### 🎯 Why We Chose It:
1. **Atomic by Design:** Redis processes commands on a single-threaded event loop. Every `HINCRBY` executes sequentially and atomically in RAM. Even if 1,000 requests arrive concurrently, zero increments are lost.
2. **Sub-Millisecond Latency:** Redis operates entirely in memory; updating a hash counter takes under 0.5 milliseconds, compared to 10–50ms for disk-backed databases.
3. **Decouples Writes from Disk:** Voting spikes never degrade MongoDB write performance because MongoDB is updated asynchronously off the critical request path.

#### ❌ Alternatives Rejected:
- **MongoDB `$inc` Operator on every vote:** While MongoDB offers atomic `$inc`, running hundreds of concurrent document updates during peak voting creates disk write queues, lock contention, and increases tail latency ($p99$).
- **Relational SQL (`UPDATE options SET votes = votes + 1`):** Row-level locking under high concurrency causes transaction serialization queues and potential deadlocks.
- **Application Memory (Go mutex in RAM):** Storing counts in a Go server variable (`sync.Mutex`) fails as soon as the backend scales horizontally to multiple server instances. Redis acts as a shared, centralized in-memory counter across all backend nodes.

---

### Decision 3: Duplicate Vote Prevention: Redis Sets (`SADD`) vs Database Lookups vs Client Cookies

#### ❓ Interview Question:
> *"How do you prevent audience members from voting multiple times, and how do you ensure the deduplication check doesn't become a bottleneck?"*

#### ✅ Decision Taken:
We use **Redis Sets (`SADD poll:<id>:voters <voter_key>`)** as the gatekeeper for duplicate votes.

#### 🎯 Why We Chose It:
1. **$O(1)$ Time Complexity:** Redis Sets use internal hash tables. Checking if an element exists and adding it takes constant time $O(1)$ regardless of whether there are 10 or 100,000 voters.
2. **Atomic Test-and-Set:** `SADD` returns `1` if the value was successfully added (first-time voter), and `0` if the value already existed (duplicate voter). This eliminates the classic race condition where two simultaneous requests from the same user both pass a "check" step before either reaches a "save" step.
3. **Zero Disk I/O:** The deduplication decision is made entirely in RAM in microseconds.

#### ❌ Alternatives Rejected:
- **MongoDB Unique Index Lookup:** Querying MongoDB (`findOne({ pollId, voterKey })`) before inserting adds disk read latency to every single vote request.
- **Client-Side Cookies or `localStorage` Only:** Trivially bypassed by opening an Incognito/Private window or clearing browser data. Client-side state is useful for UI hints, but backend enforcement is mandatory.

---

### Decision 4: Storage Strategy: Dual-Store (Redis + MongoDB) vs Single Database

#### ❓ Interview Question:
> *"Why did you use both Redis AND MongoDB? Couldn't MongoDB or Redis alone handle everything?"*

#### ✅ Decision Taken:
We adopted a **dual-store hybrid architecture**: Redis acts as the **hot in-memory state engine and message broker**, while MongoDB acts as the **cold, durable document persistence layer**.

#### 🎯 Why We Chose It:
1. **Separation of Concerns:**
   - **Redis (Hot State):** In-memory counters, instant deduplication sets, and Pub/Sub message channels.
   - **MongoDB (Cold/Durable State):** User accounts, hashed credentials, poll questions, variable-length option arrays, and audit logs.
2. **Document Model Fit:** Polls have nested structures (a question embeds an arbitrary list of options). Document databases represent this natively without table joins.
3. **Resilience & Durability:** In-memory stores like Redis can lose volatile data if an unpersisted instance crashes. MongoDB ensures poll configurations and historical vote audit trails are safely written to durable disk.

#### ❌ Alternatives Rejected:
- **Redis Only:** While Redis has RDB/AOF persistence, it is fundamentally an in-memory data store. Storing full user collections, complex relational queries, and long-term historical records in RAM is expensive and architecturally ill-suited for document querying.
- **MongoDB Only:** MongoDB alone cannot provide atomic single-thread microsecond counters without disk contention, nor does it provide a lightweight native Pub/Sub broker for driving SSE streams to clients without polling.

---

### Decision 5: Voter Identity: Salted SHA-256 IP Hashing vs Mandatory Login vs Raw IPs

#### ❓ Interview Question:
> *"How do you track voters fairly when anyone with the link can vote without logging in?"*

#### ✅ Decision Taken:
We support a **dual-identifier system**:
1. If the voter has an active JWT session $\rightarrow$ Identifier is `user:<UserID>`.
2. If the voter is an anonymous guest $\rightarrow$ Identifier is `ip:<SHA256(ClientIP + Salt)>`.

#### 🎯 Why We Chose It:
1. **Frictionless Audience Experience:** Live audiences in a meeting or presentation will not register or log in just to answer a 10-second poll. Anonymous voting is essential for high participation rates.
2. **Privacy & Security Compliance:** Storing raw IP addresses in plain text in logs violates privacy best practices (such as GDPR). Hashing the IP with a secret server salt produces a deterministic one-way fingerprint without storing the user's actual IP address.
3. **Fairness:** Authenticated users cannot vote multiple times by switching networks, and guests cannot vote twice from the same network connection.

#### ❌ Alternatives Rejected:
- **Mandatory User Registration for Voters:** Would destroy audience conversion and usability during live presentations.
- **Raw IP Address Storage:** Poses privacy risks and data liability.
- **Browser Fingerprinting (`FingerprintJS`):** Overly complex, fragile across browser updates, and blocked by modern privacy protections like Safari ITP.

---

### Decision 6: Backend Technology: Go (Gin) vs Node.js (Express) vs Python (FastAPI)

#### ❓ Interview Question:
> *"Why did you choose Go and the Gin framework for this backend?"*

#### ✅ Decision Taken:
We built the backend service in **Go using the Gin Web Framework**.

#### 🎯 Why We Chose It:
1. **Native Concurrency (Goroutines):** Go's runtime manages lightweight threads (goroutines) that consume only ~2KB of memory each. A single Go instance can easily handle tens of thousands of concurrent long-lived SSE connections with minimal CPU and RAM usage.
2. **Radix Tree Routing:** Gin uses an efficient radix-tree routing algorithm, making it one of the fastest web frameworks across any language.
3. **Compiled Single Binary:** Go compiles into a standalone binary with zero runtime dependencies (no Node interpreter, no virtual environments), simplifying deployment and Docker container footprints.
4. **Strong Typing & Safety:** Go catches type mismatches, missing fields, and nil pointers at compile time.

#### ❌ Alternatives Rejected:
- **Node.js / Express:** While Node.js handles I/O well, JavaScript's single-threaded event loop can suffer latency hiccups under CPU-heavy tasks (like bcrypt password hashing).
- **Python / FastAPI:** Python's Global Interpreter Lock (GIL) and higher memory overhead make handling thousands of concurrent open streaming sockets less resource-efficient than Go.

---

### Decision 7: Authentication Architecture: Stateless JWT vs Stateful Server Sessions

#### ❓ Interview Question:
> *"Why did you choose JWT over traditional server-side session cookies for creator authentication?"*

#### ✅ Decision Taken:
We chose **Stateless JSON Web Tokens (JWT) signed with HMAC-SHA256**.

#### 🎯 Why We Chose It:
1. **Stateless Scalability:** The server does not need to store or look up session IDs on every request. All user identity claims (`userID`, `email`) are cryptographically sealed in the token itself.
2. **CORS & Multi-Client Flexibility:** JWTs passed via the `Authorization: Bearer <token>` header avoid browser third-party cookie blocking issues across cross-origin deployments (e.g., frontend on Vercel, backend on Render).

#### ❌ Alternatives Rejected:
- **Server-Side Session Store in Redis:** Adds a Redis read lookup on every single authenticated API request. JWTs can be verified entirely with CPU cryptography in Go without touching any database.

---

### Decision 8: Durability Strategy: Async MongoDB Persistence vs Synchronous Two-Phase Writes

#### ❓ Interview Question:
> *"What happens to MongoDB when a vote is cast? Does the voter wait for MongoDB to finish writing?"*

#### ✅ Decision Taken:
We implemented an **asynchronous write-behind pattern**: the HTTP response and SSE broadcast are triggered immediately after Redis acknowledges the vote; durable persistence to MongoDB is dispatched to a background Go goroutine.

#### 🎯 Why We Chose It:
1. **Zero-Latency Responses:** The voting user receives a `200 OK` in $<5\text{ms}$ because they only wait for Redis RAM operations.
2. **Fault Isolation:** Even if MongoDB experiences a temporary write-lock or latency spike, audience members voting in real time do not experience lag.

#### ⚠️ Trade-offs & Mitigations:
- *Risk:* If the Go server crashes within the few milliseconds before the goroutine executes, the detailed audit log in MongoDB could be lost.
- *Mitigation:* Because Redis already has the incremented tally stored in RAM and synced to disk, the aggregate count remains 100% accurate.

---

### Decision 9: Frontend Framework: React (Vite SPA) vs Next.js (SSR) vs Vanilla JS

#### ❓ Interview Question:
> *"Why did you choose React with Vite rather than Next.js or Vanilla JavaScript?"*

#### ✅ Decision Taken:
We built the frontend as a **React Single Page Application (SPA) bundled with Vite**.

#### 🎯 Why We Chose It:
1. **Reactive State for Live Charts:** Live polling requires frequent, dynamic DOM updates as vote numbers and chart bar widths change. React’s virtual DOM efficiently re-renders only the changed percentage elements without full page repaints.
2. **Instant Developer Experience:** Vite provides sub-second Hot Module Replacement (HMR) and optimized ES-module bundling.
3. **Clean Decoupling:** Server-Side Rendering (Next.js) offers little benefit for a private live dashboard and dynamic live polling screen where all data updates on the client side via SSE.

---

### Decision 10: Local Infrastructure: Docker Compose vs Native Host Installations

#### ❓ Interview Question:
> *"Why did you use Docker Compose for MongoDB and Redis rather than installing them directly on your machine?"*

#### ✅ Decision Taken:
We containerized both databases using **Docker Compose (`docker-compose.yml`)**.

#### 🎯 Why We Chose It:
1. **Zero Host Machine Pollution:** Leaves macOS completely clean with no lingering background daemons or registry entries.
2. **Version Determinism:** Locks in MongoDB v7 and Redis v7 Alpine across every developer machine and CI/CD environment.
3. **One-Command Lifecycle:** `docker compose up -d` boots both isolated services with persistent volume storage; `docker compose down` halts them cleanly.
