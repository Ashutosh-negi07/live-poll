# Frontend Architecture Documentation (`frontend/`)

## 1. Overview
The frontend is a modern, responsive Single Page Application (SPA) built with **React** and **Vite**. It provides an intuitive interface for poll creators to authenticate, design polls, and monitor real-time votes, while offering audience members a streamlined, zero-friction voting screen with animated live results.

---

## 2. Key Technology Choices
- **Build Tool:** Vite (instant Hot Module Replacement, optimized ES module bundling).
- **Styling:** Modern CSS with custom properties, glassmorphism, responsive grid layouts, and smooth animations.
- **Real-Time Updates:** Native browser `EventSource` API connecting to the Go backend's Server-Sent Events (SSE) endpoint.
- **Charts / Visualizations:** Dynamic animated progress bars and bar charts illustrating vote distributions in real time without screen flickers.

---

## 3. Core Pages & Component Hierarchy

```
frontend/src/
├── components/
│   ├── Navbar.jsx          # Top navigation with user status and logout
│   ├── PollCard.jsx        # Summary card for user dashboard
│   ├── VoteChart.jsx       # Animated real-time bar chart updating on SSE
│   └── ProtectedRoute.jsx  # Guards routes requiring authenticated JWT
├── pages/
│   ├── Home.jsx            # Landing page explaining the tool + CTAs
│   ├── Login.jsx           # User sign-in form
│   ├── Register.jsx        # User sign-up form
│   ├── Dashboard.jsx       # Creator dashboard listing existing polls & analytics
│   ├── CreatePoll.jsx      # Multi-option poll creation wizard
│   └── PollView.jsx        # Live audience voting & real-time results screen
├── services/
│   ├── api.js              # Axios or fetch wrapper with JWT interceptors
│   └── sse.js              # EventSource listener handling live poll streams
├── App.jsx                 # Routing with React Router
└── main.jsx                # React root mount
```

---

## 4. Real-Time SSE Listener Pattern

```javascript
// Example: Listening to live poll updates in React
useEffect(() => {
  const eventSource = new EventSource(`http://localhost:8080/api/polls/${pollId}/stream`);

  eventSource.addEventListener("vote", (event) => {
    const data = JSON.parse(event.data);
    // Update React state: triggers smooth chart re-render with NO page refresh
    setLiveVotes(data.votes);
    setTotalVotes(data.total_votes);
  });

  eventSource.onerror = (err) => {
    console.error("SSE connection error:", err);
    // Browser EventSource automatically attempts reconnection
  };

  return () => {
    // Clean up connection when user navigates away
    eventSource.close();
  };
}, [pollId]);
```

---

## 5. Audience Voting UX
1. Audience navigates to `/poll/:id` (via shareable link or scanned QR code).
2. The UI fetches the current poll details and immediately establishes an SSE stream.
3. The user taps their chosen option.
4. The frontend issues a `POST /api/polls/:id/vote`.
5. The button transitions to a "Voted" confirmation state.
6. The SSE stream receives the updated count and updates the animated live chart in real time.
