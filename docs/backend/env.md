# Environment Configuration (`backend/.env` & `backend/.env.example`)

## What is an `.env` file?
A plain-text file containing `KEY=VALUE` pairs that configure an application for a specific environment (local dev, staging, production) without changing any source code.

## Why Not Hardcode Values in Code?
| Problem | Example |
|:---|:---|
| Security | A hardcoded `JWT_SECRET` gets committed to GitHub and is publicly visible |
| Flexibility | If you hardcode `localhost:27017`, you need to recompile Go to deploy to Mongo Atlas |
| Team collaboration | Each developer may run databases on different ports |

## The `.gitignore` Rule
```gitignore
backend/.env   ← Git never tracks this file
```
`backend/.env.example` **is** committed — it shows teammates what variables to create, with dummy values.

---

## Variable Reference

| Variable | Local Value | Purpose |
|:---|:---|:---|
| `PORT` | `8080` | TCP port the Go HTTP server listens on |
| `GIN_MODE` | `debug` | `debug` = verbose logs + stack traces. `release` = silent + fast |
| `CLIENT_ORIGIN` | `http://localhost:5173` | Whitelisted origin for CORS (our Vite React dev server) |
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string (matches `docker-compose.yml`) |
| `MONGO_DB_NAME` | `livepoll` | Database name inside MongoDB |
| `REDIS_ADDR` | `localhost:6379` | Redis host:port (matches `docker-compose.yml`) |
| `REDIS_PASSWORD` | *(empty)* | Redis auth password — empty for local dev |
| `REDIS_DB` | `0` | Redis logical database index (0–15; `0` is standard) |
| `JWT_SECRET` | `super_secret_...` | HMAC-SHA256 signing key for JSON Web Tokens |
| `JWT_EXPIRY_HOURS` | `24` | Hours before a login token expires |

## How `godotenv` Loads It
When `godotenv.Load()` runs at server startup, it reads `backend/.env` line by line and calls `os.Setenv(key, value)` for each pair. After that, `os.Getenv("PORT")` returns `"8080"` for the rest of the process lifetime.

If no `.env` file exists (as in production cloud deployments), the function returns an error which we intentionally ignore — the platform (Render, Railway, Fly.io) injects the variables directly into the process environment instead.
