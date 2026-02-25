<script>
  export let logs = [];

  function clearLogs() {
    logs = [];
  }

  async function openLogsFolder() {
    await window.go.main.App.OpenLogsFolder();
  }
</script>

<div class="logs-container">
  <div class="header">
    <h2>Logs</h2>
    <div class="actions">
      <button class="btn secondary" on:click={clearLogs}>Clear</button>
      <button class="btn secondary" on:click={openLogsFolder}>Open Logs Folder</button>
    </div>
  </div>

  <div class="log-viewer">
    {#if logs.length === 0}
      <p class="empty">No logs yet...</p>
    {:else}
      {#each logs as log}
        <div class="log-entry" class:error={log.error}>
          <span class="time">{new Date(log.time).toLocaleTimeString()}</span>
          <span class="message">{log.message}</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .logs-container {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .actions {
    display: flex;
    gap: 0.5rem;
  }

  .log-viewer {
    flex: 1;
    background: #0a0a0f;
    border-radius: 8px;
    padding: 1rem;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 0.85rem;
    overflow-y: auto;
    max-height: calc(100vh - 200px);
  }

  .empty {
    color: var(--text-secondary);
    font-style: italic;
  }

  .log-entry {
    display: flex;
    gap: 1rem;
    padding: 0.25rem 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.03);
  }

  .log-entry.error {
    color: var(--error);
  }

  .time {
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .message {
    word-break: break-word;
  }

  .btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-weight: 500;
    font-size: 0.9rem;
  }

  .btn.secondary {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-primary);
  }

  .btn:hover {
    filter: brightness(1.1);
  }
</style>

