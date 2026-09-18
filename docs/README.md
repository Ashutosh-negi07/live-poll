# Live Polling Tool — Documentation Index

Welcome to the comprehensive documentation suite for the **Real-Time Live Polling Tool**.

Every file, architectural layer, and design pattern in this project is documented in depth to help you master the underlying concepts for technical interviews and your video walkthrough.

---

## 📚 Table of Contents

### 1. System Architecture & High-Level Design
- [Architectural Decisions & Interview Q&A](file:///Users/blue/Documents/guviProject/live-poll/docs/decisions.md) — Detailed rationale for all 10 system, database, real-time, and concurrency design choices (and why alternatives were rejected).
- [Full-Stack System Architecture](file:///Users/blue/Documents/guviProject/live-poll/docs/architecture.md) — 4-layer overview, sequence diagrams, SSE vs WebSockets rationale, and the Redis atomic counter strategy.
- [Docker Compose Documentation](file:///Users/blue/Documents/guviProject/live-poll/docs/docker-compose.md) — Multi-container local infrastructure (MongoDB 7 & Redis 7 Alpine).

---

### 2. Backend Service (`backend/`)
- [Go Modules & Dependencies (`go.mod`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/go.mod.md) — Dependency management, checksum verification, and why each package was chosen.
- [Environment Configuration (`.env`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/env.md) — 12-factor configuration, security secrets, and environment variables.
- [Config Loader (`config.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/config/config.md) — Type-safe struct parsing, fallback defaults, and package mechanics.
- [Server Entry Point (`main.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/main.md) — HTTP bootstrap, CORS setup, goroutines, and graceful shutdown.

#### Database Connections (`backend/db/`)
- [MongoDB Client (`mongo.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/db/mongo.md) — Connection pooling, ping verification, and BSON collections.
- [Redis Client (`redis.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/db/redis.md) — High-throughput cache pool, ping command, and fail-fast startup.

#### Domain Models (`backend/models/`)
- [User Model (`user.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/models/user.md) — User structs, BSON tags, password hash masking, and validation.
- [Poll Model (`poll.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/models/poll.md) — Poll, Option, and VoteLog schemas, plus Redis live update payloads.

#### Middleware (`backend/middleware/`)
- [JWT Authentication (`auth.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/middleware/auth.md) — Bearer token extraction, HMAC-SHA256 signature verification, and context claims.
- [CORS Security (`cors.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/middleware/cors.md) — Cross-Origin Resource Sharing, preflight `OPTIONS`, and allowed origins.

#### HTTP & Streaming Handlers (`backend/handlers/`)
- [Authentication Handlers (`auth.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/handlers/auth.md) — Registration, bcrypt hashing, and login token issuance.
- [Poll Handlers (`poll.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/handlers/poll.md) — Poll creation, dual-store synchronization, and creator management.
- [Vote Handler (`vote.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/handlers/vote.md) — Atomic Redis `HINCRBY`, duplicate prevention with `SADD`, and async MongoDB durability.
- [Real-Time Stream Handler (`stream.go`)](file:///Users/blue/Documents/guviProject/live-poll/docs/backend/handlers/stream.md) — Server-Sent Events (SSE), Redis Pub/Sub subscription, and zero-refresh broadcasts.

---

### 3. Frontend Application (`frontend/`)
- [Frontend Architecture (`frontend/`)](file:///Users/blue/Documents/guviProject/live-poll/docs/frontend/README.md) — React + Vite SPA structure, SSE listener integration, and audience UX.
