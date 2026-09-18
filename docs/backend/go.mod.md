# Go Modules Documentation (`backend/go.mod` & `backend/go.sum`)

## What is `go.mod`?
`go.mod` is the dependency manifest for a Go project — equivalent to `package.json` in Node.js. It declares:
1. The **module path** — the unique identity of this project used in import statements.
2. The **Go version** being targeted.
3. All **direct and indirect dependencies** with their exact versions.

## What is `go.sum`?
`go.sum` is the lockfile — equivalent to `package-lock.json`. It contains SHA-256 cryptographic hashes of every dependency version. Before compiling, Go verifies these hashes to guarantee the downloaded code has not been tampered with.

## Key Difference from Node.js
In Node.js, `npm install` creates a massive `node_modules/` folder inside each project. In Go, dependencies are downloaded once to a **global module cache** (`$GOPATH/pkg/mod`) and shared across every project on your machine. Your project folder stays clean.

---

## Our `go.mod` After Phase 1

```
module github.com/Ashutosh-negi07/live-poll

go 1.27.1

require (
    github.com/gin-contrib/cors v1.7.8
    github.com/gin-gonic/gin v1.12.0
    github.com/golang-jwt/jwt/v5 v5.3.1
    github.com/joho/godotenv v1.5.1
    github.com/redis/go-redis/v9 v9.22.0
    go.mongodb.org/mongo-driver/v2 v2.9.1
    golang.org/x/crypto v0.57.0
)
```

## The 7 Direct Dependencies

| Package | Role |
|:---|:---|
| `github.com/gin-gonic/gin` | HTTP router, JSON binding, request validation |
| `github.com/gin-contrib/cors` | Cross-Origin Resource Sharing middleware |
| `github.com/joho/godotenv` | Loads `.env` file into `os.Getenv()` |
| `go.mongodb.org/mongo-driver/v2` | Official MongoDB driver (BSON, queries, collections) |
| `github.com/redis/go-redis/v9` | Redis client (HINCRBY, SADD, Pub/Sub) |
| `github.com/golang-jwt/jwt/v5` | Signs and verifies JSON Web Tokens |
| `golang.org/x/crypto` | Provides `bcrypt` for password hashing |

## Commands Used
```bash
# Fetch all 7 packages and their transitive dependencies
go get github.com/gin-gonic/gin github.com/gin-contrib/cors \
       github.com/joho/godotenv go.mongodb.org/mongo-driver/v2/mongo \
       github.com/redis/go-redis/v9 github.com/golang-jwt/jwt/v5 \
       golang.org/x/crypto/bcrypt

# Scan source files, remove unused deps, add missing indirect deps
go mod tidy
```
