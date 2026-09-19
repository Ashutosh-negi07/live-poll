import { Link } from 'react-router-dom'
import Navbar from '../components/Navbar'

export default function Landing() {
  return (
    <div>
      <Navbar />
      <div className="hero">
        <h1>Create Polls. <span>Watch Results Live.</span></h1>
        <p>Share a link — your audience votes and sees results update in real time. No account needed to vote.</p>
        <div className="hero-actions">
          <Link to="/register" className="btn btn-primary">Get Started →</Link>
          <Link to="/login" className="btn btn-ghost">Sign In</Link>
        </div>
      </div>

      <div className="features">
        <div className="card feature-card">
          <h3>⚡ Real-Time Results</h3>
          <p>Vote counts update live for every viewer via Server-Sent Events — no refresh needed.</p>
        </div>
        <div className="card feature-card">
          <h3>🔒 One Vote Per Device</h3>
          <p>Deduplication enforced atomically in Redis. No double-voting.</p>
        </div>
        <div className="card feature-card">
          <h3>🔗 Instant Sharing</h3>
          <p>Every poll gets a public link. Audience votes without needing an account.</p>
        </div>
      </div>
    </div>
  )
}
