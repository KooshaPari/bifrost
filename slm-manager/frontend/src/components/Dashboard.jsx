import React, { useState, useRef, useEffect } from 'react'

const go = window.go?.main?.App

export default function Dashboard({ status, onRefresh }) {
  const services = status?.services || []
  const [contextMenu, setContextMenu] = useState(null)
  const [platform, setPlatform] = useState({})

  const startAll = () => go?.StartAll()
  const stopAll = () => go?.StopAll()
  const restartAll = () => go?.RestartServices()
  const startBackend = () => go?.StartBackend()
  const stopBackend = () => go?.StopBackend()
  const startSLM = () => go?.StartSLMServer()
  const stopSLM = () => go?.StopSLMServer()
  const checkUpdates = () => go?.CheckForUpdates()
  const copyURL = () => go?.CopyServerURL()
  const openLogs = () => go?.OpenLogsFolder()
  const openConfig = () => go?.OpenConfigFolder()

  // Load platform info
  useEffect(() => {
    const loadPlatform = async () => {
      if (go) {
        const p = await go.GetPlatformInfo()
        if (p) setPlatform(p)
      }
    }
    loadPlatform()
  }, [])

  // Close context menu on click outside
  useEffect(() => {
    const handleClick = () => setContextMenu(null)
    document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [])

  const isBackendService = (svc) => {
    return svc.name.includes('MLX') || svc.name.includes('vLLM') || svc.name.includes('Backend')
  }

  const isRemote = (svc) => svc.name.startsWith('Remote:')

  return (
    <div style={styles.container}>
      {/* Mode indicator */}
      <div style={styles.modeBar}>
        <span style={styles.modeBadge}>
          {platform.mode === 'client' ? '🌐 Client Mode' : '🖥️ Host Mode'}
        </span>
        {platform.mode === 'host' && (
          <span style={styles.backendBadge}>
            {platform.backend === 'mlx' ? '🍎 MLX Backend' : '⚡ vLLM Backend'}
          </span>
        )}
        <span style={styles.platformBadge}>
          {platform.os}/{platform.arch}
        </span>
      </div>

      {/* Action bar */}
      <div style={styles.actions}>
        <button className="btn btn-primary" onClick={startAll}>▶ Start All</button>
        <button className="btn btn-danger" onClick={stopAll}>⏹ Stop All</button>
        <button className="btn" onClick={restartAll}>🔄 Restart</button>
        <button className="btn" onClick={onRefresh}>↻ Refresh</button>
        <div style={{ flex: 1 }} />
        <button className="btn" onClick={copyURL} title="Copy server URL to clipboard">📋 Copy URL</button>
        <button className="btn" onClick={checkUpdates}>⬆ Update</button>
      </div>

      {/* Service cards */}
      <div style={styles.grid}>
        {services.map(svc => (
          <ServiceCard
            key={svc.name}
            service={svc}
            platform={platform}
            onStart={isRemote(svc) ? null : (isBackendService(svc) ? startBackend : startSLM)}
            onStop={isRemote(svc) ? null : (isBackendService(svc) ? stopBackend : stopSLM)}
            onContextMenu={(e, svc) => {
              e.preventDefault()
              if (!isRemote(svc)) {
                setContextMenu({ x: e.clientX, y: e.clientY, service: svc, platform })
              }
            }}
          />
        ))}
      </div>

      {/* Quick links */}
      <div style={styles.quickLinks}>
        <button className="btn btn-link" onClick={openConfig}>📁 Config Folder</button>
        <button className="btn btn-link" onClick={openLogs}>📂 Logs Folder</button>
      </div>

      {/* Quick info */}
      <div style={styles.info}>
        {platform.mode === 'client' ? (
          <span>🌐 Connected to remote SLM server. Configure in Settings.</span>
        ) : (
          <span>💡 Right-click on service cards for more options. Use the system tray menu when minimized.</span>
        )}
      </div>

      {/* Native-style context menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          service={contextMenu.service}
          platform={contextMenu.platform}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  )
}

function ContextMenu({ x, y, service, platform, onClose }) {
  const menuRef = useRef(null)
  const isBackend = service.name.includes('MLX') || service.name.includes('vLLM') || service.name.includes('Backend')

  const handleAction = (action) => {
    action()
    onClose()
  }

  return (
    <div
      ref={menuRef}
      style={{ ...styles.contextMenu, left: x, top: y }}
      onClick={(e) => e.stopPropagation()}
    >
      <div style={styles.menuTitle}>{service.name}</div>
      <div style={styles.menuDivider} />

      {service.running ? (
        <>
          <button
            style={styles.menuItem}
            onClick={() => handleAction(isBackend ? () => go?.StopBackend() : () => go?.StopSLMServer())}
          >
            ⏹ Stop Service
          </button>
          <button
            style={styles.menuItem}
            onClick={() => handleAction(() => {
              if (isBackend) { go?.StopBackend(); setTimeout(() => go?.StartBackend(), 1000) }
              else { go?.StopSLMServer(); setTimeout(() => go?.StartSLMServer(), 1000) }
            })}
          >
            🔄 Restart Service
          </button>
        </>
      ) : (
        <button
          style={styles.menuItem}
          onClick={() => handleAction(isBackend ? () => go?.StartBackend() : () => go?.StartSLMServer())}
        >
          ▶ Start Service
        </button>
      )}

      <div style={styles.menuDivider} />

      <button style={styles.menuItem} onClick={() => handleAction(() => go?.OpenLogsFolder())}>
        📂 View Logs
      </button>
      <button style={styles.menuItem} onClick={() => handleAction(() => {
        go?.OpenURL(`http://localhost:${service.port}`)
      })}>
        🌐 Open in Browser
      </button>
    </div>
  )
}

function ServiceCard({ service, platform, onStart, onStop, onContextMenu }) {
  const { name, running, port, startedAt } = service
  const isRemote = name.startsWith('Remote:')

  return (
    <div
      className="card"
      style={{ ...styles.card, borderColor: running ? 'var(--success)' : 'var(--border)' }}
      onContextMenu={(e) => onContextMenu && onContextMenu(e, service)}
    >
      <div style={styles.cardHeader}>
        <div className={`status-dot ${running ? 'running' : ''}`} />
        <h3 style={styles.cardTitle}>{name}</h3>
      </div>

      <div style={styles.cardBody}>
        <div style={styles.row}>
          <span style={styles.label}>{isRemote ? 'Remote Port:' : 'Port:'}</span>
          <span style={styles.value}>{port}</span>
        </div>
        <div style={styles.row}>
          <span style={styles.label}>Status:</span>
          <span style={{ ...styles.value, color: running ? 'var(--success)' : 'var(--text-secondary)' }}>
            {running ? (isRemote ? 'Connected' : 'Running') : (isRemote ? 'Disconnected' : 'Stopped')}
          </span>
        </div>
        {startedAt && !isRemote && (
          <div style={styles.row}>
            <span style={styles.label}>Started:</span>
            <span style={styles.value}>{new Date(startedAt).toLocaleTimeString()}</span>
          </div>
        )}
      </div>

      {!isRemote && (
        <div style={styles.cardActions}>
          {running ? (
            <button className="btn btn-danger btn-sm" onClick={onStop}>Stop</button>
          ) : (
            <button className="btn btn-success btn-sm" onClick={onStart}>Start</button>
          )}
        </div>
      )}
    </div>
  )
}

const styles = {
  container: { display: 'flex', flexDirection: 'column', gap: '16px' },
  actions: { display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: '12px' },
  card: { transition: 'border-color 0.2s', cursor: 'context-menu' },
  cardHeader: { display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '12px' },
  cardTitle: { fontSize: '15px', fontWeight: 500 },
  cardBody: { marginBottom: '12px' },
  row: { display: 'flex', justifyContent: 'space-between', marginBottom: '4px', fontSize: '13px' },
  label: { color: 'var(--text-secondary)' },
  value: { fontWeight: 500 },
  cardActions: { display: 'flex', gap: '8px' },
  quickLinks: { display: 'flex', gap: '8px' },
  info: {
    padding: '10px 14px',
    background: 'rgba(0,120,212,0.1)',
    borderRadius: '4px',
    fontSize: '13px',
    color: 'var(--text-secondary)',
  },
  // Native-style context menu
  contextMenu: {
    position: 'fixed',
    background: 'var(--bg-elevated)',
    border: '1px solid var(--border)',
    borderRadius: '6px',
    boxShadow: '0 4px 16px rgba(0,0,0,0.2)',
    minWidth: '180px',
    padding: '4px 0',
    zIndex: 1000,
  },
  menuTitle: {
    padding: '8px 12px',
    fontSize: '12px',
    fontWeight: 600,
    color: 'var(--text-secondary)',
  },
  menuDivider: {
    height: '1px',
    background: 'var(--border)',
    margin: '4px 0',
  },
  menuItem: {
    display: 'block',
    width: '100%',
    padding: '8px 12px',
    border: 'none',
    background: 'transparent',
    textAlign: 'left',
    fontSize: '13px',
    color: 'var(--text)',
    cursor: 'pointer',
    transition: 'background 0.1s',
  },
}

