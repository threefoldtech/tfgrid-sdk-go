<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { fade, fly } from "svelte/transition";

  const dispatch = createEventDispatcher();

  let step = 1;
  let mnemonics = "";
  let network = "main";
  let apiKey = "";
  let error = "";

  function nextStep() {
    if (step === 1 && !mnemonics) {
      error = "Please enter your mnemonics";
      return;
    }
    if (step === 2 && !apiKey) {
      error = "Please enter your Gemini API Key";
      return;
    }

    error = "";
    if (step < 3) {
      step++;
    } else {
      complete();
    }
  }

  function prevStep() {
    if (step > 1) step--;
    error = "";
  }

  function complete() {
    dispatch("complete", { mnemonics, network, apiKey });
  }
</script>

<div class="onboarding" in:fade>
  <div class="card">
    <div class="header">
      <h1>Welcome to Grid Agent</h1>
      <p>Let's get you set up with the ThreeFold Grid</p>
    </div>

    <div class="steps">
      <div class="step" class:active={step >= 1}>1</div>
      <div class="line" class:active={step >= 2}></div>
      <div class="step" class:active={step >= 2}>2</div>
      <div class="line" class:active={step >= 3}></div>
      <div class="step" class:active={step >= 3}>3</div>
    </div>

    <div class="content">
      {#if step === 1}
        <div class="step-content" in:fly={{ x: 20, duration: 300 }}>
          <h2>Grid Credentials</h2>
          <div class="form-group">
            <label for="mnemonics">Mnemonics</label>
            <input
              type="password"
              id="mnemonics"
              bind:value={mnemonics}
              placeholder="Enter your 12-24 word secret phrase..."
            />
            <p class="hint">
              Your secret phrase is stored locally and never shared.
            </p>
          </div>
          <div class="form-group">
            <label for="network">Network</label>
            <select id="network" bind:value={network}>
              <option value="main">Mainnet</option>
              <option value="test">Testnet</option>
              <option value="dev">Devnet</option>
              <option value="qa">QA</option>
            </select>
          </div>
        </div>
      {:else if step === 2}
        <div class="step-content" in:fly={{ x: 20, duration: 300 }}>
          <h2>AI Configuration</h2>
          <div class="form-group">
            <label for="apiKey">Gemini API Key</label>
            <input
              type="password"
              id="apiKey"
              bind:value={apiKey}
              placeholder="AIzaSy..."
            />
            <p class="hint">Get your key from Google AI Studio</p>
          </div>
        </div>
      {:else if step === 3}
        <div class="step-content" in:fly={{ x: 20, duration: 300 }}>
          <h2>Ready to Start?</h2>
          <p>We'll log you in and initialize the AI agent.</p>
          <div class="summary">
            <div class="summary-item">
              <span class="label">Network:</span>
              <span class="value">{network}</span>
            </div>
            <div class="summary-item">
              <span class="label">Mnemonics:</span>
              <span class="value">Provided</span>
            </div>
            <div class="summary-item">
              <span class="label">API Key:</span>
              <span class="value">Provided</span>
            </div>
          </div>
        </div>
      {/if}
    </div>

    {#if error}
      <div class="error" transition:fade>{error}</div>
    {/if}

    <div class="actions">
      {#if step > 1}
        <button class="btn secondary" on:click={prevStep}>Back</button>
      {:else}
        <div></div>
      {/if}
      <button class="btn primary" on:click={nextStep}>
        {step === 3 ? "Finish Setup" : "Next"}
      </button>
    </div>
  </div>
</div>

<style>
  .onboarding {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    background: linear-gradient(
      135deg,
      var(--bg-primary) 0%,
      var(--bg-secondary) 100%
    );
  }

  .card {
    background: var(--bg-secondary);
    padding: 2.5rem;
    border-radius: 1rem;
    width: 100%;
    max-width: 500px;
    box-shadow:
      0 20px 25px -5px rgba(0, 0, 0, 0.1),
      0 10px 10px -5px rgba(0, 0, 0, 0.04);
    border: 1px solid var(--border);
  }

  .header {
    text-align: center;
    margin-bottom: 2rem;
  }

  .header h1 {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
    color: var(--text-primary);
  }

  .header p {
    color: var(--text-secondary);
  }

  .steps {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 2rem;
  }

  .step {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    font-size: 0.875rem;
    transition: all 0.3s ease;
  }

  .step.active {
    background: var(--accent);
    color: white;
  }

  .line {
    width: 50px;
    height: 2px;
    background: var(--bg-tertiary);
    margin: 0 0.5rem;
    transition: all 0.3s ease;
  }

  .line.active {
    background: var(--accent);
  }

  .content {
    min-height: 250px;
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  input,
  select {
    width: 100%;
    padding: 0.75rem;
    border-radius: 0.5rem;
    border: 1px solid var(--border);
    background: var(--bg-primary);
    color: var(--text-primary) !important;
    font-size: 1rem;
    transition: border-color 0.2s;
    -webkit-appearance: none;
    -moz-appearance: none;
    appearance: none;
  }

  select {
    background-color: var(--bg-primary);
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23cbd5e1' d='M6 9L1 4h10z'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    padding-right: 2.5rem;
  }

  select option {
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  input:focus,
  select:focus {
    outline: none;
    border-color: var(--accent);
  }

  .hint {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-top: 0.25rem;
  }

  .summary {
    background: var(--bg-primary);
    padding: 1rem;
    border-radius: 0.5rem;
    margin-top: 1rem;
  }

  .summary-item {
    display: flex;
    justify-content: space-between;
    padding: 0.5rem 0;
    border-bottom: 1px solid var(--border);
  }

  .summary-item:last-child {
    border-bottom: none;
  }

  .error {
    color: var(--error);
    font-size: 0.875rem;
    text-align: center;
    margin-bottom: 1rem;
    padding: 0.5rem;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 0.5rem;
  }

  .actions {
    display: flex;
    justify-content: space-between;
    margin-top: 2rem;
  }

  .btn {
    padding: 0.75rem 1.5rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    border: none;
  }

  .btn.primary {
    background: var(--accent);
    color: white;
  }

  .btn.primary:hover {
    background: var(--accent-hover);
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
</style>
