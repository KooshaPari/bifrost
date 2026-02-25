<script>
  import { createEventDispatcher } from 'svelte';
  export let status = null;
  export let updateInfo = null;

  const dispatch = createEventDispatcher();

  async function startAll() {
    await window.go.main.App.StartAll();
  }
  async function stopAll() {
    await window.go.main.App.StopAll();
  }
  async function startVLLM() {
    await window.go.main.App.StartVLLM();
  }
  async function stopVLLM() {
    await window.go.main.App.StopVLLM();
  }
  async function startSLM() {
    await window.go.main.App.StartSLMServer();
  }
  async function stopSLM() {
    await window.go.main.App.StopSLMServer();
  }
  async function applyUpdate() {
    await window.go.main.App.ApplyUpdate();
  }
</script>

<div class="dashboard">
  <div class="header">
    <h1>Service Dashboard</h1>
    <div class="actions">
      <button class="btn primary" on:click={startAll}>▶ Start All</button>
      <button class="btn danger" on:click={stopAll}>⏹ Stop All</button>
      <button class="btn secondary" on:click={() => dispatch('refresh')}>🔄 Refresh</button>
    </div>
  </div>

  {#if updateInfo?.updateAvailable}
    <div class="update-banner">
      <span>🎉 Update available: v{updateInfo.latestVersion}</span>
      <button class="btn primary small" on:click={applyUpdate}>Update Now</button>
    </div>
  {/if}

  <div class="services">
    {#if status?.services}
      {#each status.services as service}
        <div class="service-card" class:running={service.running}>
          <div class="service-header">
            <span class="status-dot" class:running={service.running}></span>
            <h3>{service.name}</h3>
          </div>
          <div class="service-info">
            <p>Port: <strong>{service.port}</strong></p>
            <p>Status: <strong>{service.running ? 'Running' : 'Stopped'}</strong></p>
            {#if service.startedAt}
              <p class="started">Started: {new Date(service.startedAt).toLocaleTimeString()}</p>
            {/if}
          </div>
          <div class="service-actions">
            {#if service.name.includes('vLLM')}
              {#if service.running}
                <button class="btn danger small" on:click={stopVLLM}>Stop</button>
              {:else}
                <button class="btn success small" on:click={startVLLM}>Start</button>
              {/if}
            {:else}
              {#if service.running}
                <button class="btn danger small" on:click={stopSLM}>Stop</button>
              {:else}
                <button class="btn success small" on:click={startSLM}>Start</button>
              {/if}
            {/if}
          </div>
        </div>
      {/each}
    {:else}
      <p class="loading">Loading services...</p>
    {/if}
  </div>

  <div class="footer">
    <button class="btn secondary" on:click={() => dispatch('checkUpdates')}>Check for Updates</button>
  </div>
</div>

<style>
  .dashboard { display: flex; flex-direction: column; gap: 1.5rem; }
  .header { display: flex; justify-content: space-between; align-items: center; }
  h1 { font-size: 1.5rem; font-weight: 600; }
  .actions { display: flex; gap: 0.5rem; }
  .update-banner {
    display: flex; justify-content: space-between; align-items: center;
    padding: 1rem; background: rgba(74, 222, 128, 0.1); border-radius: 8px;
    border: 1px solid var(--success);
  }
  .services { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; }
  .service-card {
    background: var(--bg-card); border-radius: 12px; padding: 1.25rem;
    border: 1px solid rgba(255,255,255,0.05); transition: all 0.2s;
  }
  .service-card:hover { border-color: rgba(255,255,255,0.1); }
  .service-card.running { border-color: var(--success); }
  .service-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem; }
  .status-dot { width: 10px; height: 10px; border-radius: 50%; background: var(--error); }
  .status-dot.running { background: var(--success); animation: pulse 2s infinite; }
  @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
  h3 { font-size: 1.1rem; font-weight: 500; }
  .service-info { margin-bottom: 1rem; }
  .service-info p { font-size: 0.9rem; color: var(--text-secondary); margin: 0.25rem 0; }
  .service-info strong { color: var(--text-primary); }
  .started { font-size: 0.8rem !important; }
  .service-actions { display: flex; gap: 0.5rem; }
  .btn { padding: 0.6rem 1rem; border: none; border-radius: 6px; cursor: pointer; font-weight: 500; }
  .btn.small { padding: 0.4rem 0.75rem; font-size: 0.85rem; }
  .btn.primary { background: var(--accent); color: white; }
  .btn.success { background: var(--success); color: #000; }
  .btn.danger { background: var(--error); color: white; }
  .btn.secondary { background: rgba(255,255,255,0.1); color: var(--text-primary); }
  .btn:hover { filter: brightness(1.1); }
  .footer { margin-top: 1rem; }
  .loading { color: var(--text-secondary); font-style: italic; }
</style>

