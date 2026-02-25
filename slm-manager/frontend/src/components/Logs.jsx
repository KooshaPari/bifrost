import React, { useRef, useEffect } from 'react'

const go = window.go?.main?.App

export default function Logs({ logs, onClear }) {
  const scrollRef = useRef(null)

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs])

  const openFolder = () => go?.OpenLogsFolder()

  return (
    <div style={styles.container}>
      {/* Actions */}
      <div style={styles.header}>
        <span style={styles.title}>Service Logs</span>
        <div style={styles.actions}>
          <button className="btn btn-sm" onClick={onClear}>Clear</button>
          <button className="btn btn-sm" onClick={openFolder}>Open Folder</button>
        </div>
      </div>

      {/* Log viewer */}
      <div ref={scrollRef} style={styles.viewer}>
        {logs.length === 0 ? (
          <div style={styles.empty}>No logs yet...</div>
        ) : (
          logs.map((log, i) => (
            <div key={i} style={{ ...styles.line, ...(log.type === 'error' ? styles.error : {}) }}>
              <span style={styles.time}>{log.time.toLocaleTimeString()}</span>
              <span style={styles.msg}>{log.msg}</span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  title: {
    fontSize: '15px',
    fontWeight: 500,
  },
  actions: {
    display: 'flex',
    gap: '8px',
  },
  viewer: {
    flex: 1,
    background: '#0d0d12',
    borderRadius: '4px',
    padding: '12px',
    fontFamily: 'var(--font-mono)',
    fontSize: '12px',
    overflow: 'auto',
    maxHeight: 'calc(100vh - 140px)',
  },
  empty: {
    color: 'var(--text-secondary)',
    fontStyle: 'italic',
  },
  line: {
    display: 'flex',
    gap: '12px',
    padding: '2px 0',
    borderBottom: '1px solid rgba(255,255,255,0.03)',
  },
  error: {
    color: 'var(--error)',
  },
  time: {
    color: 'var(--text-secondary)',
    flexShrink: 0,
  },
  msg: {
    wordBreak: 'break-word',
  },
}

