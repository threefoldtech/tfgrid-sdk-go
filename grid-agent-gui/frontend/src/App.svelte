<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { GetSettings, SaveSettings, SendMessage, SetTheme } from '../wailsjs/go/main/App.js';
  import Onboarding from './components/Onboarding.svelte';
  import ChatInterface from './components/ChatInterface.svelte';
  import { themeStore, settingsStore, messagesStore } from './stores/stores';

  import AnsiToHtml from 'ansi-to-html';

  let isConfigured = false;
  let isLoading = true;
  let currentTheme = 'dark';
  let errorMessage = '';
  let showErrorModal = false;

  // ANSI to HTML converter for error messages
  const ansiConverter = new AnsiToHtml({
    fg: '#d4d4d4',
    bg: '#1e1e1e',
    newline: true,
    escapeXML: true,
  });

  function renderAnsi(text: string): string {
    if (!text) return '';
    return ansiConverter.toHtml(text);
  }

  onMount(async () => {
    try {
      const settings = await GetSettings();
      isConfigured = settings.isConfigured;
      currentTheme = settings.theme || 'dark';
      themeStore.set(currentTheme);
      settingsStore.set(settings);
      
      // Clear messages on startup (history not persisted across sessions)
      messagesStore.set([]);
    } catch (error) {
      console.error('Failed to load settings:', error);
    } finally {
      isLoading = false;
    }
  });

  async function handleOnboardingComplete(event: CustomEvent) {
    const { mnemonics, network, apiKey } = event.detail;
    try {
      await SaveSettings(mnemonics, network, apiKey);
      isConfigured = true;
      const settings = await GetSettings();
      settingsStore.set(settings);
    } catch (error) {
      console.error('Failed to save settings:', error);
      errorMessage = 'Failed to save settings: ' + error;
      showErrorModal = true;
    }
  }

  function closeErrorModal() {
    showErrorModal = false;
    errorMessage = '';
  }

  async function toggleTheme() {
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    currentTheme = newTheme;
    themeStore.set(newTheme);
    await SetTheme(newTheme);
  }

  $: document.documentElement.setAttribute('data-theme', currentTheme);
</script>

<main class="app" data-theme={currentTheme}>
  {#if isLoading}
    <div class="loading">
      <div class="spinner"></div>
      <p>Loading Grid Agent...</p>
    </div>
  {:else if !isConfigured}
    <Onboarding on:complete={handleOnboardingComplete} />
  {:else}
    <ChatInterface {toggleTheme} theme={currentTheme} />
  {/if}

  <!-- Error Modal -->
  {#if showErrorModal}
    <div class="modal-overlay" on:click={closeErrorModal} transition:fade>
      <div class="modal error-modal" on:click|stopPropagation transition:fade>
        <div class="error-icon">⚠️</div>
        <h2>Error</h2>
        <p class="error-text">{@html renderAnsi(errorMessage)}</p>
        <div class="modal-actions">
          <button class="btn primary" on:click={closeErrorModal}>OK</button>
        </div>
      </div>
    </div>
  {/if}
</main>

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(body) {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  :global(:root) {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --bg-tertiary: #334155;
    --text-primary: #f1f5f9;
    --text-secondary: #cbd5e1;
    --accent: #3b82f6;
    --accent-hover: #2563eb;
    --success: #10b981;
    --error: #ef4444;
    --border: #475569;
  }

  :global([data-theme="light"]) {
    --bg-primary: #ffffff;
    --bg-secondary: #f8fafc;
    --bg-tertiary: #e2e8f0;
    --text-primary: #0f172a;
    --text-secondary: #475569;
    --accent: #3b82f6;
    --accent-hover: #2563eb;
    --success: #10b981;
    --error: #ef4444;
    --border: #cbd5e1;
  }

  .app {
    width: 100vw;
    height: 100vh;
    background: var(--bg-primary);
    color: var(--text-primary);
    overflow: hidden;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    gap: 1rem;
  }

  .spinner {
    width: 50px;
    height: 50px;
    border: 4px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Error Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--bg-secondary);
    border-radius: 1rem;
    padding: 2rem;
    max-width: 400px;
    width: 90%;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border);
    text-align: center;
  }

  .error-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .modal h2 {
    margin: 0 0 1rem 0;
    color: var(--text-primary);
    font-size: 1.25rem;
  }

  .modal p {
    margin: 0 0 1.5rem 0;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .error-text {
    font-family: 'Courier New', Consolas, Monaco, monospace;
    font-size: 0.875rem;
    text-align: left;
    background: var(--bg-primary);
    padding: 1rem;
    border-radius: 0.5rem;
    overflow-x: auto;
  }

  .modal-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: center;
  }

  .btn {
    padding: 0.625rem 1.5rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
    font-size: 0.875rem;
  }

  .btn.primary {
    background: var(--accent);
    color: white;
  }

  .btn.primary:hover {
    background: var(--accent-hover);
  }
</style>
