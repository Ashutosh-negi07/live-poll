import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import Navbar from '../components/Navbar'
import api from '../api'
import { useAuth } from '../context/AuthContext'

export default function Poll() {
  const { id } = useParams()
  const { user, loading: authLoading } = useAuth()

  // Key is scoped to user account so Person 1 and Person 2 have independent voted states.
  // We wait until auth finishes loading before reading localStorage.
  const storageKey = authLoading ? null : `voted_${user?.id ?? 'anon'}_${id}`

  const [poll, setPoll] = useState(null)
  const [votes, setVotes] = useState({})
  const [total, setTotal] = useState(0)
  const [voted, setVoted] = useState(false)
  const [selected, setSelected] = useState(null)
  const [voting, setVoting] = useState(false)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)
  const [live, setLive] = useState(false)

  // Once we know which user is logged in, load their voted state from localStorage
  useEffect(() => {
    if (!storageKey) return
    setVoted(localStorage.getItem(storageKey) === 'true')
  }, [storageKey])

  // Load poll data
  useEffect(() => {
    api.get(`/polls/${id}`)
      .then(res => { setPoll(res.data); updateVotes(res.data.votes || {}) })
      .catch(() => {})
  }, [id])

  // SSE — real-time vote updates
  useEffect(() => {
    const base = import.meta.env.VITE_API_URL || 'http://localhost:8080/api'
    const es = new EventSource(`${base}/polls/${id}/stream`)

    es.addEventListener('snapshot', e => { updateVotes(JSON.parse(e.data)); setLive(true) })
    es.addEventListener('vote',     e => updateVotes(JSON.parse(e.data)))
    es.onerror = () => setLive(false)

    return () => es.close()
  }, [id])

  function updateVotes(raw) {
    const parsed = {}
    let t = 0
    for (const [k, v] of Object.entries(raw)) {
      parsed[k] = parseInt(v, 10) || 0
      t += parsed[k]
    }
    setVotes(parsed)
    setTotal(t)
  }

  async function handleVote() {
    if (!selected || voting || voted) return
    setVoting(true)
    setError('')
    try {
      await api.post(`/polls/${id}/vote`, { option_id: selected })
      setVoted(true)
      if (storageKey) localStorage.setItem(storageKey, 'true')
    } catch (err) {
      const msg = err.response?.data?.error || 'Vote failed'
      if (msg.includes('already voted')) {
        setVoted(true)
        if (storageKey) localStorage.setItem(storageKey, 'true')
      }
      setError(msg)
    } finally {
      setVoting(false)
    }
  }

  function copyLink() {
    navigator.clipboard.writeText(window.location.href)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (!poll) return <><Navbar /><div className="spinner" /></>

  const closed = !poll.is_active

  return (
    <div>
      <Navbar />
      <div className="poll-page">

        <h1 className="poll-title">{poll.title}</h1>
        {poll.description && <p className="poll-desc">{poll.description}</p>}

        {closed && <div className="alert alert-error">🔒 This poll is closed.</div>}
        {voted  && <div className="alert alert-success">✅ You have voted on this poll.</div>}
        {error  && <div className="alert alert-error">{error}</div>}

        {/* Voting form — only when not yet voted and poll is open */}
        {!voted && !closed && (
          <>
            {poll.options.map(opt => (
              <button
                key={opt.id}
                id={`vote-${opt.id}`}
                className={`option-btn ${selected === opt.id ? 'selected' : ''}`}
                onClick={() => setSelected(opt.id)}
              >
                <span>{opt.text}</span>
                {selected === opt.id && <span>✓</span>}
              </button>
            ))}
            <button
              id="submit-vote"
              className="btn btn-primary"
              style={{ width: '100%', marginTop: '.5rem' }}
              onClick={handleVote}
              disabled={!selected || voting}
            >
              {voting ? 'Submitting…' : 'Submit Vote'}
            </button>
            <div className="divider" />
          </>
        )}

        {/* Results — always visible */}
        <div className="chart-section">
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', marginBottom: '1rem' }}>
            {live && <span className="live-badge" style={{ marginRight: '0.5rem' }}>● LIVE</span>}
            {total} vote{total !== 1 ? 's' : ''}
          </p>
          {poll.options.map(opt => {
            const count = votes[opt.id] || 0
            const pct = total > 0 ? Math.round((count / total) * 100) : 0
            return (
              <div key={opt.id} className="chart-row">
                <div className="chart-label">
                  <span>{opt.text}</span>
                  <span style={{ color: 'var(--muted)' }}>{count} ({pct}%)</span>
                </div>
                <div className="chart-bar-bg">
                  <div className="chart-bar" style={{ width: `${pct}%` }} />
                </div>
              </div>
            )
          })}
        </div>

        <div className="share-box">
          <input readOnly value={window.location.href} />
          <button id="copy-link" className="btn btn-ghost btn-sm" onClick={copyLink}>
            {copied ? '✓ Copied' : 'Copy'}
          </button>
        </div>

        <div style={{ marginTop: '1.5rem' }}>
          <Link to="/" className="btn btn-ghost btn-sm">← Home</Link>
        </div>
      </div>
    </div>
  )
}
