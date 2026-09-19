import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import Navbar from '../components/Navbar'
import { useAuth } from '../context/AuthContext'
import api from '../api'

export default function Dashboard() {
  const { user } = useAuth()
  const [polls, setPolls] = useState([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [options, setOptions] = useState(['', ''])

  useEffect(() => {
    api.get('/polls/my')
      .then(res => setPolls(res.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleCreate = async (e) => {
    e.preventDefault()
    setError(''); setSuccess('')
    const filled = options.filter(o => o.trim())
    if (filled.length < 2) { setError('Add at least 2 options.'); return }
    setCreating(true)
    try {
      const res = await api.post('/polls', { title, description, options: filled })
      setPolls([res.data, ...polls])
      setSuccess('Poll created!')
      setTitle(''); setDescription(''); setOptions(['', ''])
      setTimeout(() => setSuccess(''), 3000)
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to create poll.')
    } finally {
      setCreating(false)
    }
  }

  const handleClose = async (pollId, e) => {
    e.preventDefault(); e.stopPropagation()
    if (!confirm('Close this poll? Voters will no longer be able to vote.')) return
    try {
      await api.patch(`/polls/${pollId}/close`)
      setPolls(polls.map(p => p.id === pollId ? { ...p, is_active: false } : p))
    } catch { alert('Failed to close.') }
  }

  const handleDelete = async (pollId, e) => {
    e.preventDefault(); e.stopPropagation()
    if (!confirm('Delete this poll permanently? This cannot be undone.')) return
    try {
      await api.delete(`/polls/${pollId}`)
      setPolls(polls.filter(p => p.id !== pollId))
    } catch { alert('Failed to delete.') }
  }

  return (
    <div>
      <Navbar />
      <div className="page">
        <p style={{ color: 'var(--muted)', marginBottom: '1.25rem', fontSize: '.9rem' }}>
          Welcome back, <strong>{user?.name}</strong>
        </p>

        <div className="dashboard-grid">
          {/* Create Poll */}
          <div className="card" style={{ height: 'fit-content' }}>
            <p className="page-title">New Poll</p>
            {error && <div className="alert alert-error">{error}</div>}
            {success && <div className="alert alert-success">{success}</div>}
            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label className="form-label">Question</label>
                <input id="poll-title" className="form-input" placeholder="Your question…" value={title} onChange={e => setTitle(e.target.value)} required />
              </div>
              <div className="form-group">
                <label className="form-label">Description (optional)</label>
                <input id="poll-desc" className="form-input" placeholder="Extra context…" value={description} onChange={e => setDescription(e.target.value)} />
              </div>
              <div className="form-group">
                <label className="form-label">Options</label>
                {options.map((opt, i) => (
                  <div key={i} className="option-row">
                    <input id={`opt-${i}`} className="form-input" placeholder={`Option ${i + 1}`} value={opt} onChange={e => { const n = [...options]; n[i] = e.target.value; setOptions(n) }} />
                    {options.length > 2 && <button type="button" className="btn btn-danger btn-sm" onClick={() => setOptions(options.filter((_,idx) => idx !== i))}>✕</button>}
                  </div>
                ))}
                {options.length < 8 && <button type="button" className="add-option-btn" onClick={() => setOptions([...options, ''])}>+ Add option</button>}
              </div>
              <button id="create-poll-btn" className="btn btn-primary" style={{ width: '100%' }} disabled={creating}>
                {creating ? 'Creating…' : 'Create Poll'}
              </button>
            </form>
          </div>

          {/* My Polls */}
          <div>
            <p className="page-title">My Polls</p>
            {loading ? <div className="spinner" /> : polls.length === 0 ? (
              <div className="empty-state">No polls yet. Create your first one!</div>
            ) : (
              <div className="polls-list">
                {polls.map(p => (
                  <Link key={p.id} to={`/poll/${p.id}`} className="card poll-card">
                    <div>
                      <div className="poll-card-title">{p.title}</div>
                      <div className="poll-card-meta">{p.options?.length} options · {new Date(p.created_at).toLocaleDateString()}</div>
                    </div>
                    <div className="poll-actions">
                      <span className={`badge ${p.is_active ? 'badge-active' : 'badge-closed'}`}>{p.is_active ? 'Live' : 'Closed'}</span>
                      {p.is_active && <button className="btn btn-danger btn-sm" onClick={e => handleClose(p.id, e)}>Close</button>}
                      <button className="btn btn-danger btn-sm" style={{ opacity: 0.7 }} onClick={e => handleDelete(p.id, e)}>Delete</button>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
