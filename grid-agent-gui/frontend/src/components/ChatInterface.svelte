<script lang="ts">
  import { onMount, afterUpdate } from "svelte";
  import { SendMessage, Logout } from "../../wailsjs/go/main/App.js";
  import { messagesStore } from "../stores/stores";
  import ChatMessage from "./ChatMessage.svelte";
  import { fade } from "svelte/transition";
  import tfLogo from "../assets/images/tf-logo.png";
  import AnsiToHtml from "ansi-to-html";

  export let toggleTheme: () => void;
  export let theme: string;

  let input = "";
  let chatContainer: HTMLElement;
  let isSending = false;
  let showLogoutModal = false;
  let showErrorModal = false;
  let errorMessage = "";

  // ANSI to HTML converter
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

  function scrollToBottom() {
    if (chatContainer) {
      chatContainer.scrollTop = chatContainer.scrollHeight;
    }
  }

  afterUpdate(scrollToBottom);

  async function handleSubmit() {
    if (!input.trim() || isSending) return;

    const userMsg = {
      role: "user",
      content: input,
      timestamp: new Date().toISOString(),
      isCommand: false,
      output: "",
      error: "",
    };

    messagesStore.update((msgs) => [...msgs, userMsg]);
    const messageToSend = input;
    input = "";
    isSending = true;

    try {
      const response = await SendMessage(messageToSend);
      messagesStore.update((msgs) => [...msgs, response]);
    } catch (error) {
      console.error("Failed to send message:", error);
      messagesStore.update((msgs) => [
        ...msgs,
        {
          role: "agent",
          content: "Error: " + error,
          timestamp: new Date().toISOString(),
          isCommand: false,
          output: "",
          error: error.toString(),
        },
      ]);
    } finally {
      isSending = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleLogout() {
    showLogoutModal = true;
  }

  function cancelLogout() {
    showLogoutModal = false;
  }

  async function confirmLogout() {
    showLogoutModal = false;
    try {
      await Logout();
      // Clear messages
      messagesStore.set([]);
      // Reload to show onboarding
      window.location.reload();
    } catch (error) {
      console.error("Logout failed:", error);
      errorMessage = "Failed to logout: " + error;
      showErrorModal = true;
    }
  }

  function closeErrorModal() {
    showErrorModal = false;
    errorMessage = "";
  }
</script>

<div class="chat-interface" in:fade>
  <header>
    <div class="logo">
      <img src={tfLogo} alt="ThreeFold Logo" />
      <span>Grid Agent</span>
    </div>
    <div class="controls">
      <button class="icon-btn" on:click={toggleTheme} title="Toggle Theme">
        {#if theme === "dark"}
          🌙
        {:else}
          ☀️
        {/if}
      </button>
      <button class="icon-btn logout-btn" on:click={handleLogout} title="Logout">
        ⎋
      </button>
    </div>
  </header>

  <div class="messages" bind:this={chatContainer}>
    {#if $messagesStore.length === 0}
      <div class="empty-state">
        <h2>How can I help you today?</h2>
        <p>Ask me to deploy VMs, check nodes, or manage your grid resources.</p>
      </div>
    {/if}

    {#each $messagesStore as msg}
      <ChatMessage message={msg} />
    {/each}

    {#if isSending}
      <div class="typing-indicator">
        <span></span>
        <span></span>
        <span></span>
      </div>
    {/if}
  </div>

  <div class="input-area">
    <div class="input-wrapper">
      <textarea
        bind:value={input}
        on:keydown={handleKeydown}
        placeholder="Type a message..."
        rows="1"
      ></textarea>
      <button
        class="send-btn"
        on:click={handleSubmit}
        disabled={!input.trim() || isSending}
      >
        Send
      </button>
    </div>
  </div>
</div>

<!-- Logout Confirmation Modal -->
{#if showLogoutModal}
  <div class="modal-overlay" on:click={cancelLogout} transition:fade>
    <div class="modal" on:click|stopPropagation transition:fade>
      <h2>Confirm Logout</h2>
      <p>Are you sure you want to logout?</p>
      <p class="warning">This will clear your credentials and return to the setup screen.</p>
      <div class="modal-actions">
        <button class="btn secondary" on:click={cancelLogout}>Cancel</button>
        <button class="btn danger" on:click={confirmLogout}>Logout</button>
      </div>
    </div>
  </div>
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

<style>
  .chat-interface {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-weight: 600;
    font-size: 1.125rem;
  }

  .logo img {
    height: 24px;
    width: auto;
  }

  .controls {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .icon-btn {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 1.25rem;
    padding: 0.5rem;
    border-radius: 0.5rem;
    transition: background 0.2s;
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-btn:hover {
    background: var(--bg-tertiary);
  }

  .logout-btn {
    color: var(--text-secondary);
  }

  .logout-btn:hover {
    color: var(--error);
    background: rgba(239, 68, 68, 0.1);
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .empty-state {
    text-align: center;
    margin-top: 20vh;
    color: var(--text-secondary);
  }

  .empty-state h2 {
    color: var(--text-primary);
    margin-bottom: 0.5rem;
  }

  .input-area {
    padding: 1.5rem;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border);
  }

  .input-wrapper {
    display: flex;
    gap: 1rem;
    background: var(--bg-primary);
    padding: 0.75rem;
    border-radius: 0.75rem;
    border: 1px solid var(--border);
    transition: border-color 0.2s;
  }

  .input-wrapper:focus-within {
    border-color: var(--accent);
  }

  textarea {
    flex: 1;
    background: transparent;
    border: none;
    color: var(--text-primary);
    font-size: 1rem;
    resize: none;
    padding: 0.25rem;
    font-family: inherit;
  }

  textarea:focus {
    outline: none;
  }

  .send-btn {
    background: var(--accent);
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
  }

  .send-btn:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .send-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .typing-indicator {
    display: flex;
    gap: 0.25rem;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 1rem;
    width: fit-content;
    margin-left: 0;
  }

  .typing-indicator span {
    width: 8px;
    height: 8px;
    background: var(--text-secondary);
    border-radius: 50%;
    animation: bounce 1.4s infinite ease-in-out both;
  }

  .typing-indicator span:nth-child(1) {
    animation-delay: -0.32s;
  }
  .typing-indicator span:nth-child(2) {
    animation-delay: -0.16s;
  }

  @keyframes bounce {
    0%,
    80%,
    100% {
      transform: scale(0);
    }
    40% {
      transform: scale(1);
    }
  }

  /* Logout Modal */
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
  }

  .modal h2 {
    margin: 0 0 1rem 0;
    color: var(--text-primary);
    font-size: 1.25rem;
  }

  .modal p {
    margin: 0 0 0.5rem 0;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .modal .warning {
    color: var(--error);
    font-size: 0.875rem;
    margin-bottom: 1.5rem;
  }

  .modal-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
  }

  .btn {
    padding: 0.625rem 1.25rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
    font-size: 0.875rem;
  }

  .btn.secondary {
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .btn.secondary:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .btn.danger {
    background: var(--error);
    color: white;
  }

  .btn.danger:hover {
    background: #dc2626;
  }

  /* Error Modal */
  .error-modal {
    text-align: center;
  }

  .error-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
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

  .btn.primary {
    background: var(--accent);
    color: white;
  }

  .btn.primary:hover {
    background: var(--accent-hover);
  }
</style>
