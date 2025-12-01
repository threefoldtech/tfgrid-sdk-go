<script lang="ts">
  import { fade } from "svelte/transition";
  import CommandOutput from "./CommandOutput.svelte";
  import logo from "../assets/images/tf-logo.png";
  import AnsiToHtml from "ansi-to-html";
  import { marked } from "marked";
  import DOMPurify from "dompurify";

  export let message: {
    role: string;
    content: string;
    timestamp: string;
    requestID?: string;
    steps?: Array<{
      type: string;
      content: string;
      output: string;
      error: string;
    }>;
    // Deprecated fields (backward compatibility)
    isCommand?: boolean;
    output?: string;
    error?: string;
  };

  const isUser = message.role === "user";
  let showSteps = false;

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

  // Render Markdown to HTML
  function renderMarkdown(text: string): string {
    if (!text) return "";

    // Configure marked options
    // breaks: true ensures that single newlines are converted to <br>,
    // which is better for "normal text" that isn't strictly markdown formatted.
    // gfm: true enables GitHub Flavored Markdown (tables, etc.)
    marked.setOptions({
      gfm: true,
      breaks: true,
    });

    // marked.parse returns a string or Promise<string>. In sync mode (default), it's string.
    const rawHtml = marked.parse(text) as string;
    return DOMPurify.sanitize(rawHtml);
  }
</script>

<div class="message-wrapper {isUser ? 'user' : 'agent'}" in:fade>
  <div class="avatar">
    {#if isUser}
      👤
    {:else}
      <img src={logo} alt="Agent" />
    {/if}
  </div>

  <div class="content-wrapper">
    <div class="bubble">
      {#if message.content}
        <div class="text markdown-body">
          {@html renderMarkdown(message.content)}
        </div>
      {:else if message.steps && message.steps.length > 0}
        <div class="typing-indicator">
          <span></span>
          <span></span>
          <span></span>
        </div>
      {:else if message.role === "agent"}
        <div class="typing-indicator">
          <span></span>
          <span></span>
          <span></span>
        </div>
      {/if}
    </div>

    <!-- New: Steps display (Option 3 - Rich Message) -->
    {#if message.steps && message.steps.length > 0}
      <div class="steps-container">
        <button class="steps-toggle" on:click={() => (showSteps = !showSteps)}>
          <span class="toggle-icon">{showSteps ? "▼" : "▶"}</span>
          Show workflow ({message.steps.length}
          {message.steps.length === 1 ? "step" : "steps"})
        </button>

        {#if showSteps}
          <div class="steps" transition:fade>
            {#each message.steps as step, i}
              <div class="step">
                <div class="step-header">
                  <span class="step-number">{i + 1}</span>
                  <span class="step-type">
                    {#if step.type === "command"}
                      ⚡ Command Executed
                    {:else if step.type === "url_fetch"}
                      🌐 URL Fetched
                    {:else if step.type === "analysis"}
                      📊 Analysis
                    {:else if step.type === "question"}
                      ❓ Question
                    {:else}
                      🔄 Processing
                    {/if}
                  </span>
                </div>

                <div class="step-content">
                  <div class="step-command">{step.content}</div>

                  {#if step.output}
                    <div class="step-output">
                      <div class="output-label">
                        {step.type === "url_fetch"
                          ? "📄 Content:"
                          : "📤 Output:"}
                      </div>
                      <pre class="ansi-output">{@html renderAnsi(
                          step.output,
                        )}</pre>
                    </div>
                  {/if}

                  {#if step.error}
                    <div class="step-error">
                      <div class="error-label">❌ Error:</div>
                      <pre class="ansi-output">{@html renderAnsi(
                          step.error,
                        )}</pre>
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Backward compatibility: Old command display -->
    {#if message.isCommand && !message.steps}
      <CommandOutput output={message.output} error={message.error} />
    {/if}

    {#if message.content}
      <div class="timestamp">
        {new Date(message.timestamp).toLocaleTimeString()}
      </div>
    {/if}
  </div>
</div>

<style>
  .message-wrapper {
    display: flex;
    gap: 1rem;
    max-width: 80%;
  }

  .message-wrapper.user {
    margin-left: auto;
    flex-direction: row-reverse;
  }

  .avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--bg-tertiary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.25rem;
    flex-shrink: 0;
    overflow: hidden;
  }

  .avatar img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .content-wrapper {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .bubble {
    padding: 1rem;
    border-radius: 1rem;
    background: var(--bg-secondary);
    color: var(--text-primary);
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    overflow-wrap: anywhere;
    text-align: left;
  }

  .user .bubble {
    background: var(--accent);
    color: white;
    border-bottom-right-radius: 0.25rem;
  }

  .agent .bubble {
    border-top-left-radius: 0.25rem;
  }

  .timestamp {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin: 0 0.5rem;
  }

  .user .timestamp {
    text-align: right;
  }

  /* Typing Indicator */
  .typing-indicator {
    display: flex;
    gap: 0.25rem;
    padding: 0.25rem 0;
  }

  .typing-indicator span {
    width: 6px;
    height: 6px;
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

  /* Steps styling */
  .steps-container {
    margin-top: 0.75rem;
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
  }

  .steps-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    justify-content: space-between;
  }

  .steps-toggle {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.875rem;
    padding: 0.25rem 0;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    transition: color 0.2s;
  }

  .steps-toggle:hover {
    color: var(--text-primary);
  }

  .abort-btn {
    background: var(--error);
    color: white;
    border: none;
    padding: 0.375rem 0.75rem;
    border-radius: 0.375rem;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .abort-btn:hover:not(:disabled) {
    background: #dc2626;
    transform: translateY(-1px);
  }

  .abort-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .toggle-icon {
    font-size: 0.75rem;
  }

  .steps {
    margin-top: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .step {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 0.5rem;
    padding: 0.75rem;
  }

  .step-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .step-number {
    background: var(--accent);
    color: white;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
  }

  .step-type {
    font-size: 0.875rem;
  }

  .step-content {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .step-command {
    font-family: "Courier New", monospace;
    background: var(--bg-primary);
    padding: 0.5rem;
    border-radius: 0.25rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    white-space: pre-wrap;
    word-break: break-all;
    overflow-wrap: anywhere;
    text-align: left;
    overflow-x: auto;
  }

  .step-output,
  .step-error {
    margin-top: 0.25rem;
  }

  .output-label,
  .error-label {
    font-size: 0.75rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
    color: var(--text-secondary);
  }

  .error-label {
    color: #ef4444;
  }

  .step-output pre,
  .step-error pre {
    background: var(--bg-primary);
    padding: 0.5rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    overflow-x: auto;
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
    overflow-wrap: anywhere;
    color: var(--text-secondary);
    text-align: left;
  }

  .step-error pre {
    color: #ef4444;
  }

  /* ANSI output styling */
  .ansi-output {
    font-family: "Courier New", Consolas, Monaco, monospace;
    line-height: 1.4;
  }

  /* Override ansi-to-html default styles to match our theme */
  .ansi-output :global(span) {
    font-family: inherit;
  }

  /* Markdown Styling */
  .markdown-body :global(p) {
    margin-bottom: 0.5rem;
  }

  .markdown-body :global(p:last-child) {
    margin-bottom: 0;
  }

  .markdown-body :global(ul),
  .markdown-body :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.5rem;
  }

  .markdown-body :global(li) {
    margin-bottom: 0.25rem;
  }

  .markdown-body :global(pre) {
    background: var(--bg-primary);
    padding: 0.75rem;
    border-radius: 0.5rem;
    overflow-x: auto;
    margin: 0.5rem 0;
  }

  .markdown-body :global(code) {
    font-family: "Courier New", monospace;
    background: rgba(0, 0, 0, 0.2);
    padding: 0.1rem 0.3rem;
    border-radius: 0.25rem;
    font-size: 0.9em;
  }

  .markdown-body :global(pre) :global(code) {
    background: transparent;
    padding: 0;
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .markdown-body :global(a) {
    color: var(--accent);
    text-decoration: none;
  }

  .markdown-body :global(a:hover) {
    text-decoration: underline;
  }

  .markdown-body :global(strong) {
    font-weight: 600;
    color: var(--text-primary);
  }
</style>
