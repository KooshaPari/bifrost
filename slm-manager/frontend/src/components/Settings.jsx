import React, { useState, useEffect } from 'react'

const go = window.go?.main?.App

export default function Settings() {
  const [config, setConfig] = useState({
    mode: 'host',
    backend: 'mlx',
    backendHost: 'localhost',
    backendPort: 8000,
    slmPort: 8081,
    model: 'mlx-community/Qwen2.5-3B-Instruct-4bit',
    remoteHost: '',
    remotePort: 8081,
    autoStartAll: true,
    launchAtLogin: false,
    repoOwner: 'kooshapari',
    repoName: 'bifrost-extensions',
  })
  const [version, setVersion] = useState({})
  const [platform, setPlatform] = useState({})
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const load = async () => {
      if (go) {
        const c = await go.GetConfig()
        if (c) setConfig(c)
        const v = await go.GetVersionInfo()
        if (v) setVersion(v)
        const p = await go.GetPlatformInfo()
        if (p) setPlatform(p)
      }
    }
    load()
  }, [])

  const save = async () => {
    setSaving(true)
    await go?.SaveConfig(config)
    setSaving(false)
  }

  const update = (key, value) => {
    setConfig(prev => ({ ...prev, [key]: value }))
  }

  const toggleLaunchAtLogin = async (enabled) => {
    update('launchAtLogin', enabled)
    // Immediately apply the launch at login change
    await go?.SetLaunchAtLogin(enabled)
  }

  const openFolder = (type) => {
    if (type === 'config') go?.OpenConfigFolder()
    else if (type === 'logs') go?.OpenLogsFolder()
  }

  return (
    <div style={styles.container}>
      {/* Mode Selection */}
      <section className="card" style={styles.section}>
        <h3 style={styles.sectionTitle}>Mode</h3>
        <div style={styles.modeSelector}>
          <label style={styles.modeOption}>
            <input
              type="radio"
              name="mode"
              checked={config.mode === 'host'}
              onChange={() => update('mode', 'host')}
            />
            <div>
              <strong>🖥️ Host Mode</strong>
              <p style={styles.modeDesc}>Run local inference ({platform.supportsMLX ? 'MLX' : 'vLLM'})</p>
            </div>
          </label>
          <label style={styles.modeOption}>
            <input
              type="radio"
              name="mode"
              checked={config.mode === 'client'}
              onChange={() => update('mode', 'client')}
            />
            <div>
              <strong>🌐 Client Mode</strong>
              <p style={styles.modeDesc}>Connect to remote SLM server</p>
            </div>
          </label>
        </div>
      </section>

      {/* Host Mode Settings */}
      {config.mode === 'host' && (
        <>
          <section className="card" style={styles.section}>
            <h3 style={styles.sectionTitle}>Backend Configuration</h3>
            {platform.supportsMLX && platform.supportsVLLM && (
              <div style={styles.field}>
                <label style={styles.label}>Backend</label>
                <select
                  value={config.backend}
                  onChange={(e) => update('backend', e.target.value)}
                  style={styles.select}
                >
                  <option value="mlx">MLX (Apple Silicon)</option>
                  <option value="vllm">vLLM</option>
                </select>
              </div>
            )}
            <Field label="Host" value={config.backendHost} onChange={v => update('backendHost', v)} />
            <Field label="Port" type="number" value={config.backendPort} onChange={v => update('backendPort', +v)} />
            <Field label="Model" value={config.model} onChange={v => update('model', v)} />
            <p style={styles.hint}>
              {config.backend === 'mlx'
                ? '💡 Use MLX models from HuggingFace (e.g., mlx-community/Qwen2.5-3B-Instruct-4bit)'
                : '💡 Use standard HuggingFace model IDs (e.g., Qwen/Qwen2.5-3B-Instruct)'}
            </p>
          </section>

          <section className="card" style={styles.section}>
            <h3 style={styles.sectionTitle}>SLM Server</h3>
            <Field label="Port" type="number" value={config.slmPort} onChange={v => update('slmPort', +v)} />
          </section>
        </>
      )}

      {/* Client Mode Settings */}
      {config.mode === 'client' && (
        <section className="card" style={styles.section}>
          <h3 style={styles.sectionTitle}>Remote Server</h3>
          <Field
            label="Remote Host"
            value={config.remoteHost}
            onChange={v => update('remoteHost', v)}
            placeholder="e.g., 192.168.1.100 or my-server.local"
          />
          <Field label="Remote Port" type="number" value={config.remotePort} onChange={v => update('remotePort', +v)} />
          <p style={styles.hint}>
            💡 Connect to your Windows/Linux server running SLM Server (e.g., your RTX 3090 Ti machine)
          </p>
        </section>
      )}

      {/* Startup Section */}
      <section className="card" style={styles.section}>
        <h3 style={styles.sectionTitle}>Startup</h3>
        <Checkbox
          label="Launch at login"
          checked={config.launchAtLogin}
          onChange={toggleLaunchAtLogin}
        />
        <Checkbox
          label="Auto-start all services on launch"
          checked={config.autoStartAll}
          onChange={v => update('autoStartAll', v)}
        />
      </section>

      {/* Files Section */}
      <section className="card" style={styles.section}>
        <h3 style={styles.sectionTitle}>Files</h3>
        <div style={styles.buttonRow}>
          <button className="btn" onClick={() => openFolder('config')}>📁 Open Config Folder</button>
          <button className="btn" onClick={() => openFolder('logs')}>📂 Open Logs Folder</button>
        </div>
      </section>

      {/* Updates Section */}
      <section className="card" style={styles.section}>
        <h3 style={styles.sectionTitle}>Updates</h3>
        <Field label="GitHub Owner" value={config.repoOwner} onChange={v => update('repoOwner', v)} />
        <Field label="GitHub Repo" value={config.repoName} onChange={v => update('repoName', v)} />
        <button className="btn" onClick={() => go?.CheckForUpdates()} style={{ marginTop: '8px' }}>
          Check for Updates
        </button>
      </section>

      {/* Save button */}
      <button className="btn btn-primary" onClick={save} disabled={saving} style={{ marginTop: '8px' }}>
        {saving ? 'Saving...' : 'Save Settings'}
      </button>

      {/* About */}
      <section style={styles.about}>
        <div style={styles.aboutRow}><span>Version:</span> <strong>{version.version}</strong></div>
        <div style={styles.aboutRow}><span>Commit:</span> <strong>{version.commit?.substring(0, 7)}</strong></div>
        <div style={styles.aboutRow}><span>Platform:</span> <strong>{version.os}/{version.arch}</strong></div>
        <div style={styles.footer}>
          <span>SLM Manager is part of </span>
          <a href="#" onClick={(e) => { e.preventDefault(); go?.OpenURL('https://github.com/kooshapari/bifrost-extensions') }}>
            Bifrost Extensions
          </a>
          <span> | License: MIT</span>
        </div>
      </section>
    </div>
  )
}

function Field({ label, value, onChange, type = 'text', placeholder }) {
  return (
    <div style={styles.field}>
      <label style={styles.label}>{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        style={styles.input}
      />
    </div>
  )
}

function Checkbox({ label, checked, onChange }) {
  return (
    <label style={styles.checkbox}>
      <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)} />
      <span>{label}</span>
    </label>
  )
}

const styles = {
  container: { maxWidth: '500px' },
  section: { marginBottom: '16px' },
  sectionTitle: { fontSize: '14px', color: 'var(--accent)', marginBottom: '12px', fontWeight: 500 },
  modeSelector: { display: 'flex', flexDirection: 'column', gap: '8px' },
  modeOption: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: '10px',
    padding: '10px',
    border: '1px solid var(--border)',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'background 0.1s',
  },
  modeDesc: { fontSize: '12px', color: 'var(--text-secondary)', margin: '4px 0 0' },
  field: { marginBottom: '10px' },
  label: { display: 'block', fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '4px' },
  input: { width: '100%', padding: '8px 10px' },
  select: { width: '100%', padding: '8px 10px', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: '4px', color: 'var(--text)' },
  hint: { fontSize: '11px', color: 'var(--text-secondary)', margin: '8px 0 0', padding: '8px', background: 'rgba(0,120,212,0.1)', borderRadius: '4px' },
  checkbox: { display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', fontSize: '13px', marginBottom: '8px' },
  buttonRow: { display: 'flex', gap: '8px', flexWrap: 'wrap' },
  about: { marginTop: '20px', padding: '12px', background: 'rgba(0,0,0,0.2)', borderRadius: '4px', fontSize: '12px' },
  aboutRow: { marginBottom: '4px', color: 'var(--text-secondary)' },
  footer: { marginTop: '8px', paddingTop: '8px', borderTop: '1px solid var(--border)', color: 'var(--text-secondary)', fontSize: '11px' },
}

