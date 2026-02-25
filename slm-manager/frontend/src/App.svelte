<script>
  import { onMount } from 'svelte';
  import Dashboard from './components/Dashboard.svelte';
  import Settings from './components/Settings.svelte';
  import Logs from './components/Logs.svelte';

  let activeTab = 'dashboard';
  let status = null;
  let logs = [];
  let updateInfo = null;

  onMount(async () => {
    // Load initial status
    await refreshStatus();

    // Listen for events from Go backend
    window.runtime.EventsOn('serviceUpdate', (data) => {
      status = data;
    });

    window.runtime.EventsOn('log', (message) => {
      logs = [...logs, { time: new Date().toISOString(), message }].slice(-100);
    });

    window.runtime.EventsOn('error', (message) => {
      logs = [...logs, { time: new Date().toISOString(), message: `❌ ${message}`, error: true }].slice(-100);
    });

    // Poll for status every 5 seconds
    setInterval(refreshStatus, 5000);
  });

  async function refreshStatus() {
    try {
      status = await window.go.main.App.GetStatus();
    } catch (e) {
      console.error('Failed to get status:', e);
    }
  }

  async function checkUpdates() {
    try {
      updateInfo = await window.go.main.App.CheckForUpdates();
    } catch (e) {
      console.error('Failed to check updates:', e);
    }
  }
</script>

<main>
  <nav>
    <div class="logo">
      <span class="icon">🚀</span>
      <span>SLM Manager</span>
    </div>
    <div class="tabs">
      <button class:active={activeTab === 'dashboard'} on:click={() => activeTab = 'dashboard'}>
        Dashboard
      </button>
      <button class:active={activeTab === 'logs'} on:click={() => activeTab = 'logs'}>
        Logs
      </button>
      <button class:active={activeTab === 'settings'} on:click={() => activeTab = 'settings'}>
        Settings
      </button>
    </div>
    <div class="version">
      {#if status}v{status.version}{/if}
    </div>
  </nav>

  <div class="content">
    {#if activeTab === 'dashboard'}
      <Dashboard {status} {updateInfo} on:checkUpdates={checkUpdates} on:refresh={refreshStatus} />
    {:else if activeTab === 'logs'}
      <Logs {logs} />
    {:else if activeTab === 'settings'}
      <Settings />
    {/if}
  </div>
</main>

<style>
  main { display: flex; flex-direction: column; height: 100vh; }
  nav {
    display: flex; align-items: center; justify-content: space-between;
    padding: 1rem 1.5rem; background: var(--bg-secondary);
    border-bottom: 1px solid rgba(255,255,255,0.1);
  }
  .logo { display: flex; align-items: center; gap: 0.5rem; font-weight: 600; font-size: 1.1rem; }
  .logo .icon { font-size: 1.4rem; }
  .tabs { display: flex; gap: 0.5rem; }
  .tabs button {
    padding: 0.5rem 1rem; border: none; border-radius: 6px;
    background: transparent; color: var(--text-secondary);
    cursor: pointer; font-size: 0.9rem; transition: all 0.2s;
  }
  .tabs button:hover { background: rgba(255,255,255,0.05); color: var(--text-primary); }
  .tabs button.active { background: var(--accent); color: white; }
  .version { font-size: 0.8rem; color: var(--text-secondary); }
  .content { flex: 1; padding: 1.5rem; overflow-y: auto; }
</style>

