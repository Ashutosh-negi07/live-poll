# Docker Compose Documentation (`docker-compose.yml`)

## 1. What is Docker & Docker Compose?
- **Docker:** A tool that runs applications in lightweight, isolated environments called **containers**. A container packages the software, libraries, and runtime together so it behaves identically on any machine.
- **Docker Compose:** A tool for defining and running multi-container Docker applications using a single configuration file (`docker-compose.yml`). Instead of starting MongoDB and Redis separately with long terminal commands, Docker Compose manages both together.

---

## 2. Why are we using it in this project?
1. **Zero host pollution:** You do not need to install MongoDB or Redis directly onto your macOS system.
2. **Version control:** Guarantees you are running MongoDB v7 and Redis v7 Alpine, eliminating "it works on my machine" issues.
3. **One-command lifecycle:**
   - `docker compose up -d` starts both databases.
   - `docker compose down` cleanly stops them.
4. **Data persistence:** Even if you restart or stop the containers, your database entries stay intact inside Docker volumes.

---

## 3. Detailed Breakdown of `docker-compose.yml`

```yaml
version: "3.8"

services:
  # ----------------------------------------------------
  # Service 1: MongoDB (Primary persistent database)
  # ----------------------------------------------------
  mongodb:
    image: mongo:7
    container_name: poll_mongo
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
    environment:
      MONGO_INITDB_DATABASE: livepoll

  # ----------------------------------------------------
  # Service 2: Redis (Real-time pub/sub & atomic counter)
  # ----------------------------------------------------
  redis:
    image: redis:7-alpine
    container_name: poll_redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

# ------------------------------------------------------
# Named Storage Volumes (Host storage managed by Docker)
# ------------------------------------------------------
volumes:
  mongo_data:
  redis_data:
```

---

## 4. Deep Dive: How Each Component Functions

### A. The MongoDB Service (`mongodb`)
- **`image: mongo:7`**: Tells Docker to pull the official MongoDB version 7 image from Docker Hub.
- **`container_name: poll_mongo`**: Assigns a human-readable name to the running container so you can inspect it with `docker logs poll_mongo` or access its shell.
- **`ports: "27017:27017"`**:
  - The format is `HOST_PORT:CONTAINER_PORT`.
  - Maps port `27017` on your Mac to port `27017` inside the container.
  - Allows your Go backend running on your Mac to connect to `mongodb://localhost:27017`.
- **`volumes: mongo_data:/data/db`**:
  - By default, files created inside a container are deleted when the container is removed.
  - This line mounts a named volume (`mongo_data`) to MongoDB's internal data directory (`/data/db`).
  - Your polls, users, and vote records persist safely across container restarts.
- **`environment: MONGO_INITDB_DATABASE: livepoll`**:
  - Automatically initializes a database named `livepoll` when the container runs for the first time.

### B. The Redis Service (`redis`)
- **`image: redis:7-alpine`**: Alpine Linux is an ultra-lightweight distribution (around 5MB base). This gives you a fast, minimal Redis instance.
- **`container_name: poll_redis`**: Friendly name for running Redis CLI or viewing logs.
- **`ports: "6379:6379"`**:
  - Maps port `6379` on your Mac to port `6379` in the container.
  - Allows the Go backend to connect to `localhost:6379`.
- **`volumes: redis_data:/data`**:
  - Stores Redis snapshots (`dump.rdb`) so in-memory state can be restored if the container restarts.

### C. The Volumes Block (`volumes:`)
- Declares top-level persistent volumes (`mongo_data` and `redis_data`).
- Docker stores these on your Mac's disk under Docker's virtual storage path, completely isolated from your project workspace.

---

## 5. Roles in the Architecture

| Database | Role in this Application | Why this specific tool? |
| :--- | :--- | :--- |
| **MongoDB** | Primary persistence store | Stores structured documents like users (credentials, profile) and polls (question, options, metadata). Poll documents vary in structure (different numbers of options), making a document store a natural fit. |
| **Redis** | Real-time state & message bus | Stores live vote tallies using atomic counters (`HINCRBY`) to prevent race conditions during high voting traffic, and acts as the Pub/Sub broker to push live vote events to connected clients via Server-Sent Events (SSE). |

---

## 6. Essential Commands Cheat Sheet

```bash
# Start both containers in the background (detached mode)
docker compose up -d

# Check status of running containers
docker ps

# View live logs of a specific container
docker compose logs -f mongodb
docker compose logs -f redis

# Open an interactive MongoDB shell inside the container
docker exec -it poll_mongo mongosh

# Open an interactive Redis CLI inside the container
docker exec -it poll_redis redis-cli

# Stop containers without losing database records
docker compose down

# Stop containers AND delete stored data (clean slate)
docker compose down -v
```
