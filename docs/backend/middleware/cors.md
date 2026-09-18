# CORS Middleware Documentation (`backend/middleware/cors.go`)

## 1. What is CORS?
CORS stands for **Cross-Origin Resource Sharing**. By default, web browsers enforce a strict security rule called the **Same-Origin Policy**, which prevents JavaScript executing on one origin (e.g., `http://localhost:5173`) from reading responses from a different origin (e.g., `http://localhost:8080`).

---

## 2. Why is it used in this project?
In a modern decoupled architecture:
- Frontend runs on Vite dev server: `http://localhost:5173`
- Backend runs on Go HTTP server: `http://localhost:8080`

Because the port numbers differ, browsers classify these as different origins. Without CORS configuration, every `fetch` or `axios` call from React to Go fails with browser network errors:
```
Access to fetch at 'http://localhost:8080/api/...' has been blocked by CORS policy.
```
Gin CORS middleware adds the required HTTP response headers so the browser permits cross-origin API calls.

---

## 3. How CORS Works Internally

### A. Preflight `OPTIONS` Requests
For requests that alter data (`POST`, `PUT`, `DELETE`) or include custom headers (like `Authorization: Bearer <token>`), the browser automatically sends a lightweight "preflight" request with the HTTP `OPTIONS` method before sending the actual request.
- The browser asks: *"Are you willing to accept a POST request with an Authorization header from origin http://localhost:5173?"*
- The Go backend inspects the origin and replies with:
  ```http
  Access-Control-Allow-Origin: http://localhost:5173
  Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
  Access-Control-Allow-Headers: Authorization, Content-Type
  Access-Control-Allow-Credentials: true
  ```
- Only upon receiving these headers does the browser transmit the actual `POST` payload.

---

## 4. Configuration Breakdown

```go
cors.New(cors.Config{
    AllowOrigins: []string{
        cfg.ClientOrigin,       // Configured in .env (http://localhost:5173)
        "http://localhost:3000",// Alternative local dev port
    },
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour, // Caches preflight approval for 12 hours
})
```

- **`AllowOrigins`**: Strictly lists trusted domains rather than using `*` (wildcard), which is insecure when credentials/tokens are transmitted.
- **`AllowHeaders`**: Explicitly permits the `Authorization` header containing our JWT.
- **`AllowCredentials`**: Permits browser cookies and authorization headers across origins.
- **`MaxAge`**: Caches the preflight check result in the client browser, eliminating redundant `OPTIONS` requests and speeding up subsequent API calls.
