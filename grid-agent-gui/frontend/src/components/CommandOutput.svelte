<script lang="ts">
  import { slide } from "svelte/transition";
  import AnsiToHtml from "ansi-to-html";

  export let output: string;
  export let error: string;

  let isExpanded = false;
  const hasError = !!error;

  function toggle() {
    isExpanded = !isExpanded;
  }

  // ANSI to HTML converter
  const ansiConverter = new AnsiToHtml({
    fg: "#d4d4d4",
    bg: "#1e1e1e",
    newline: true,
    escapeXML: true,
  });

  // Convert ANSI codes to HTML
  function renderAnsi(text: string): string {
    if (!text) return "";
    return ansiConverter.toHtml(text);
  }
</script>

<div class="command-output" class:error={hasError}>
  <button class="header" on:click={toggle}>
    <span class="status-icon">{hasError ? "❌" : "✅"}</span>
    <span class="title"
      >Command Execution {hasError ? "Failed" : "Success"}</span
    >
    <span class="chevron" class:expanded={isExpanded}>▼</span>
  </button>

  {#if isExpanded}
    <div class="details" transition:slide>
      {#if output}
        <div class="section">
          <div class="label">Output:</div>
          <pre class="ansi-output"><code>{@html renderAnsi(output)}</code></pre>
        </div>
      {/if}

      {#if error}
        <div class="section error-section">
          <div class="label">Error:</div>
          <pre class="ansi-output"><code>{@html renderAnsi(error)}</code></pre>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .command-output {
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    overflow: hidden;
    margin-top: 0.5rem;
    font-size: 0.9rem;
  }

  .command-output.error {
    border-color: var(--error);
  }

  .header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.75rem;
    background: transparent;
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    text-align: left;
  }

  .header:hover {
    background: var(--bg-tertiary);
  }

  .status-icon {
    font-size: 1rem;
  }

  .title {
    flex: 1;
    font-weight: 500;
  }

  .chevron {
    transition: transform 0.2s;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .chevron.expanded {
    transform: rotate(180deg);
  }

  .details {
    padding: 0 0.75rem 0.75rem;
    border-top: 1px solid var(--border);
  }

  .section {
    margin-top: 0.75rem;
  }

  .label {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-bottom: 0.25rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  pre {
    background: var(--bg-tertiary);
    padding: 0.75rem;
    border-radius: 0.25rem;
    overflow-x: auto;
    font-family: "Fira Code", monospace;
    font-size: 0.85rem;
    color: var(--text-primary);
    white-space: pre-wrap;
    word-break: break-all;
    overflow-wrap: anywhere;
    text-align: left;
  }

  .error-section pre {
    background: rgba(239, 68, 68, 0.1);
    color: var(--error);
  }

  /* ANSI output styling */
  .ansi-output {
    font-family: "Courier New", Consolas, Monaco, monospace;
    line-height: 1.4;
  }

  .ansi-output :global(span) {
    font-family: inherit;
  }
</style>
