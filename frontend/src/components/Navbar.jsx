import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Navbar() {
  const { user, logout } = useAuth()

  return (
    <nav className="navbar">
      <Link to="/" className="navbar-brand">⚡ LivePoll</Link>
      <div className="navbar-links">
        {user ? (
          <>
            <span style={{ color: 'var(--text-muted)', fontSize: '.875rem' }}>
              Hi, {user.name}
            </span>
            <Link to="/dashboard" className="btn btn-ghost btn-sm">Dashboard</Link>
            <button className="btn btn-danger btn-sm" onClick={logout}>Logout</button>
          </>
        ) : (
          <>
            <Link to="/login" className="btn btn-ghost btn-sm">Login</Link>
            <Link to="/register" className="btn btn-primary btn-sm">Sign Up</Link>
          </>
        )}
      </div>
    </nav>
  )
}
