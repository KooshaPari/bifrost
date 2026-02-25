<script>
  import { onMount } from 'svelte';

  let config = {
    vllmHost: 'localhost',
    vllmPort: 8000,
    slmPort: 8081,
    vllmModel: 'Qwen/Qwen2.5-3B-Instruct',
    autoStartAll: true,
    repoOwner: 'kooshapari',
    repoName: 'bifrost-extensions'
  };

  let versionInfo = {};
  let saving = false;
  let saved = false;

  onMount(async () => {
    try {
      config = await window.go.main.App.GetConfig();
      versionInfo = await window.go.main.App.GetVersionInfo();
    } catch (e) {
      console.error('Failed to load config:', e);
    }
  });

  async function saveConfig() {
    saving = true;
    try {
      await window.go.main.App.SaveConfig(config);
      saved = true;
      setTimeout(() => saved = false, 2000);
    } catch (e) {
      console.error('Failed to save config:', e);
    }
    saving = false;
  }
</script>

<div class="settings">
  <h2>Settings</h2>

  <div class="section">
    <h3>vLLM Configuration</h3>
    <div class="form-group">
      <label for="vllmHost">Host</label>
      <input id="vllmHost" type="text" bind:value={config.vllmHost} />
    </div>
    <div class="form-group">
      <label for="vllmPort">Port</label>
      <input id="vllmPort" type="number" bind:value={config.vllmPort} />
    </div>
    <div class="form-group">
      <label for="vllmModel">Model</label>
      <input id="vllmModel" type="text" bind:value={config.vllmModel} />
    </div>
  </div>

  <div class="section">
    <h3>SLM Server Configuration</h3>
    <div class="form-group">
      <label for="slmPort">Port</label>
      <input id="slmPort" type="number" bind:value={config.slmPort} />
    </div>
  </div>

  <div class="section">
    <h3>Startup</h3>
    <div class="form-group checkbox">
      <input id="autoStart" type="checkbox" bind:checked={config.autoStartAll} />
      <label for="autoStart">Auto-start all services on launch</label>
    </div>
  </div>

  <div class="section">
    <h3>Updates</h3>
    <div class="form-group">
      <label for="repoOwner">GitHub Owner</label>
      <input id="repoOwner" type="text" bind:value={config.repoOwner} />
    </div>
    <div class="form-group">
      <label for="repoName">GitHub Repo</label>
      <input id="repoName" type="text" bind:value={config.repoName} />
    </div>
  </div>

  <div class="actions">
    <button class="btn primary" on:click={saveConfig} disabled={saving}>
      {saving ? 'Saving...' : saved ? '✓ Saved!' : 'Save Settings'}
    </button>
  </div>

  <div class="version-info">
    <h3>About</h3>
    <p>Version: <strong>{versionInfo.version || 'dev'}</strong></p>
    <p>Commit: <strong>{versionInfo.commit || 'unknown'}</strong></p>
    <p>Build Date: <strong>{versionInfo.buildDate || 'unknown'}</strong></p>
    <p>Platform: <strong>{versionInfo.os}/{versionInfo.arch}</strong></p>
  </div>
</div>

<style>
  .settings { max-width: 600px; }
  h2 { font-size: 1.3rem; margin-bottom: 1.5rem; }
  .section { margin-bottom: 2rem; padding: 1.25rem; background: var(--bg-card); border-radius: 10px; }
  h3 { font-size: 1rem; margin-bottom: 1rem; color: var(--accent); }
  .form-group { margin-bottom: 1rem; }
  .form-group.checkbox { display: flex; align-items: center; gap: 0.5rem; }
  label { display: block; font-size: 0.9rem; color: var(--text-secondary); margin-bottom: 0.4rem; }
  .checkbox label { margin-bottom: 0; cursor: pointer; }
  input[type="text"], input[type="number"] {
    width: 100%; padding: 0.6rem; background: rgba(0,0,0,0.3);
    border: 1px solid rgba(255,255,255,0.1); border-radius: 6px;
    color: var(--text-primary); font-size: 0.95rem;
  }
  input:focus { outline: none; border-color: var(--accent); }
  input[type="checkbox"] { width: 18px; height: 18px; cursor: pointer; accent-color: var(--accent); }
  .actions { margin-top: 1.5rem; }
  .btn { padding: 0.7rem 1.5rem; border: none; border-radius: 6px; cursor: pointer; font-weight: 500; }
  .btn.primary { background: var(--accent); color: white; }
  .btn:disabled { opacity: 0.6; cursor: not-allowed; }
  .version-info { margin-top: 2rem; padding: 1rem; background: rgba(0,0,0,0.2); border-radius: 8px; }
  .version-info h3 { color: var(--text-secondary); }
  .version-info p { font-size: 0.85rem; color: var(--text-secondary); margin: 0.3rem 0; }
  .version-info strong { color: var(--text-primary); }
</style>

