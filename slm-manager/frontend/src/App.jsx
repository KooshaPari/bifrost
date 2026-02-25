import React, { useState, useEffect, useCallback } from 'react'
import Dashboard from './components/Dashboard'
import Logs from './components/Logs'
import Settings from './components/Settings'

// Wails runtime bindings
const go = window.go?.main?.App

export default function App() {
  const [tab, setTab] = useState('dashboard')
  const [status, setStatus] = useState(null)
  const [logs, setLogs] = useState([])
  const [version, setVersion] = useState('')

  // Load initial data
  useEffect(() => {
    const init = async () => {
      if (go) {
        const s = await go.GetStatus()
        setStatus(s)
        setVersion(s?.version || 'dev')
      }
    }
    init()

    // Listen for events from Go backend
    if (window.runtime) {
      window.runtime.EventsOn('serviceUpdate', (data) => {
        setStatus(data)
      })
      window.runtime.EventsOn('log', (msg) => {
        setLogs(prev => [...prev.slice(-99), { time: new Date(), msg, type: 'info' }])
      })
      window.runtime.EventsOn('error', (msg) => {
        setLogs(prev => [...prev.slice(-99), { time: new Date(), msg, type: 'error' }])
      })
    }

    // Poll status every 5 seconds
    const interval = setInterval(async () => {
      if (go) {
        const s = await go.GetStatus()
        setStatus(s)
      }
    }, 5000)

    return () => clearInterval(interval)
  }, [])

  const refresh = useCallback(async () => {
    if (go) {
      const s = await go.GetStatus()
      setStatus(s)
    }
  }, [])

  return (
    <div style={styles.container}>
      {/* Minimal tab bar - native feeling */}
      <nav style={styles.nav}>
        <div style={styles.tabs}>
          {['dashboard', 'logs', 'settings'].map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              style={{
                ...styles.tab,
                ...(tab === t ? styles.tabActive : {})
              }}
            >
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>
        <span style={styles.version}>v{version}</span>
      </nav>

      {/* Content */}
      <main style={styles.main}>
        {tab === 'dashboard' && <Dashboard status={status} onRefresh={refresh} />}
        {tab === 'logs' && <Logs logs={logs} onClear={() => setLogs([])} />}
        {tab === 'settings' && <Settings />}
      </main>
    </div>
  )
}

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column',
    height: '100vh',
    background: 'var(--bg-primary)',
  },
  nav: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '8px 16px',
    background: 'var(--bg-secondary)',
    borderBottom: '1px solid var(--border)',
    // Make it draggable for window move
    WebkitAppRegion: 'drag',
  },
  tabs: {
    display: 'flex',
    gap: '4px',
    WebkitAppRegion: 'no-drag',
  },
  tab: {
    padding: '6px 14px',
    border: 'none',
    borderRadius: '4px',
    background: 'transparent',
    color: 'var(--text-secondary)',
    fontSize: '13px',
    cursor: 'pointer',
    transition: 'all 0.15s',
  },
  tabActive: {
    background: 'var(--accent)',
    color: 'white',
  },
  version: {
    fontSize: '12px',
    color: 'var(--text-secondary)',
  },
  main: {
    flex: 1,
    padding: '16px',
    overflow: 'auto',
  },
}

