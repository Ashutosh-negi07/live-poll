# Full-Stack System Architecture Documentation

## 1. Overview & Problem Statement
The goal is to build a high-performance, real-time live polling platform. A user creates a poll, shares the link with an audience, and when audience members vote, results update live for everyone watching with **zero page refresh**.

---

## 2. Technology Stack & Architectural Roles

```
                      +------------------------------------------+
                      |         React Single Page App            |
                      |  (Vite, Tailwind/CSS, Recharts, SSE)     |
                      +--------------------+---------------------+
                                           |
                    HTTP POST (Vote/Auth)  |  HTTP GET (SSE Stream)
                                           v
                      +------------------------------------------+
                      |          Go Gin Web Backend              |
                      | (Goroutines, JWT Auth, Input Validation) |
                      +--------------------+---------------------+
                                           |
                   +-----------------------+-----------------------+
                   |                                               |
                   v                                               v
+------------------------------------+   +------------------------------------+
|          Redis 7 (In-Memory)       |   |       MongoDB 7 (Document DB)      |
|                                    |   |                                    |
| • Atomic Counters (`HINCRBY`)      |   | • Persistent Users & Credentials   |
| • Voter Deduplication (`SADD`)     |   | • Poll Question & Options Metadata |
| • Pub/Sub Broadcast (`PUBLISH`)    |   | • Long-Term Vote Audit Logs        |
+------------------------------------+   +------------------------------------+
```

| Layer | Technology | Architectural Role | Why this technology? |
| :--- | :--- | :--- | :--- |
| **Frontend** | React (Vite) | Interactive user interface | Fast rendering, reactive state management, animated live vote charts, and native `EventSource` SSE client. |
| **Backend** | Go (Gin Framework) | API gateway & business logic | Ultra-fast execution, low memory footprint (~20MB RAM), native concurrency via lightweight goroutines, and strict compile-time type safety. |
| **Realtime** | Redis 7 | Live state & message broker | In-memory atomic counters (`HINCRBY`) eliminate race conditions under concurrent votes; Redis Pub/Sub drives instant SSE broadcasts to connected viewers. |
| **Database** | MongoDB 7 | Durable persistence store | Document-oriented data model naturally stores polls with variable numbers of embedded options and audit logs without complex SQL migrations. |

---

## 3. End-to-End System Workflows

### Flow 1: Poll Creation (Authenticated)
1. User logs in: Client sends credentials to `POST /api/auth/login`.
2. Backend verifies bcrypt hash in MongoDB, generates a 24-hour signed JWT, and returns it to the client.
3. User creates a poll: Client sends `POST /api/polls` with `Authorization: Bearer <token>`.
4. Go middleware validates the JWT and injects `userID` into the request context.
5. Go backend persists the poll in MongoDB, generating an `_id`.
6. Go backend seeds the Redis hash `poll:<id>:votes` with initial 0 counts for each option.
7. Backend returns the poll details and shareable URL.

---

### Flow 2: Live Voting (High Concurrency & Real-Time Broadcast)
1. An audience member clicks an option: Frontend sends `POST /api/polls/:id/vote`.
2. **Duplicate Vote Check (Redis `SADD`):**
   - Backend calculates a voter key (either `user:<UserID>` if logged in, or `ip:<SHA256(IP)>` if guest).
   - Backend executes `SADD poll:<id>:voters <voter_key>`.
   - If Redis returns `0`, the voter has already participated. Backend immediately returns `409 Conflict`.
3. **Atomic Count Increment (Redis `HINCRBY`):**
   - If `SADD` returns `1`, backend executes `HINCRBY poll:<id>:votes <option_id> 1`.
   - Redis updates the count in RAM in less than 1 millisecond.
4. **Instant Event Broadcast (Redis `PUBLISH`):**
   - Backend publishes the new vote snapshot to Redis Pub/Sub topic `poll:<id>:live`.
5. **SSE Stream Delivery (Zero Page Refresh):**
   - All open SSE connections (`GET /api/polls/:id/stream`) for that poll receive the Pub/Sub message.
   - The Go streaming handler flushes the event to all connected React clients.
   - React state updates and the chart animates immediately for every viewer in real time.
6. **Asynchronous Durability (MongoDB):**
   - In a non-blocking background goroutine, the backend records the vote log and updates the poll's `total_votes` in MongoDB.

---

## 4. Key Architectural Decisions & Interview Talking Points

### A. Why SSE Instead of WebSockets?
- **Directionality:** Live voting is fundamentally asymmetric. Voting is a discrete action (HTTP POST); results are a continuous one-way stream from server to clients.
- **Simplicity & Efficiency:** WebSockets require switching protocols (`ws://`), managing custom heartbeats, and complex proxy configuration. SSE works over standard HTTP, supports native browser reconnection (`EventSource`), and seamlessly integrates with HTTP caching and load balancers.

### B. The Hardest Technical Challenge Faced & Solved
- **The Problem: Race Conditions Under Concurrent Voting.**
  - If 500 audience members vote in the same 2 seconds, traditional database updates (`votes = votes + 1`) suffer from read-modify-write race conditions, leading to missing votes and database lock contention.
- **The Solution: Redis Atomic Operations.**
  - By offloading the hot counter to Redis `HINCRBY` (which executes atomically on Redis's single-threaded event loop) and duplicate detection to Redis `SADD`, the system guarantees 100% data accuracy at microsecond latency. MongoDB writes are deferred asynchronously.

---

## 5. Mandatory Submission Requirements Checklist
- [x] **Separation of Concerns:** `backend/` and `frontend/` are completely decoupled.
- [x] **All 4 Layers Active:** React, Go Gin, Redis, and MongoDB each perform critical, non-trivial work.
- [x] **Backend Validation:** Server-side validation on all endpoints using Gin struct binding.
- [x] **Real-Time Live Updates:** Zero page refresh updates powered by SSE and Redis Pub/Sub.
- [x] **Authentication:** Protected poll creation with JWT middleware and bcrypt password hashing.
- [ ] **Public GitHub Repo:** Pushed to GitHub with clear README and documentation.
- [ ] **Live Deployed URL:** Deployed to a publicly accessible cloud environment (e.g., Render, Fly.io, Railway, Vercel).
- [ ] **3–5 Min Walkthrough Video:** Explaining architecture, the hardest technical challenge, and AI usage, submitted to `devhiring@hclguvi.com`.
