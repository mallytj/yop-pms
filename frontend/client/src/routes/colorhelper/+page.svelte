<script lang="ts">
  import "$lib/styles/app.css";
  // 1. STATE MANAGEMENT
  let hue = 265; // Default: Purple
  let chroma = 0.22; // Default: Vibrant
  let isDark = false;

  // Scale helper
  const steps = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900];

  // Interaction: Copy variable to clipboard
  function copy(text: string) {
    navigator.clipboard.writeText(text);
    // In a real app, you'd trigger a toast notification here
  }
</script>

<div
  class="ds-page"
  class:dark-mode={isDark}
  style="
    --brand-hue: {hue}; 
    --brand-chroma: {chroma};
  "
>
  <div class="controls">
    <div class="control-group">
      <label for="hue">Hue ({hue})</label>
      <input id="hue" type="range" min="0" max="360" bind:value={hue} />
    </div>

    <div class="control-group">
      <label for="chroma">Chroma ({chroma})</label>
      <input
        id="chroma"
        type="range"
        min="0"
        max="0.3"
        step="0.01"
        bind:value={chroma}
      />
    </div>

    <div class="separator"></div>

    <button class="theme-toggle" on:click={() => (isDark = !isDark)}>
      {isDark ? "Switch to Light ☀️" : "Switch to Dark 🌙"}
    </button>
  </div>

  <main class="content-wrapper">
    <header class="page-header">
      <h1>Design System API</h1>
      <p>
        Real-time token visualizer. All colors below are generated
        mathematically from
        <span class="highlight">Hue: {hue}</span> and
        <span class="highlight">Chroma: {chroma}</span>.
      </p>
    </header>

    <section class="section">
      <h2 class="section-title">
        <span class="indicator brand"></span> Primitives
      </h2>

      <div class="grid-layout">
        <div class="ramp-container">
          <h3 class="ramp-title">Brand Palette</h3>
          <div class="swatch-grid">
            {#each steps as step}
              <button
                class="swatch"
                style="background-color: var(--palette-brand-{step})"
                on:click={() => copy(`var(--palette-brand-${step})`)}
              >
                <span class="swatch-label">{step}</span>
              </button>
            {/each}
          </div>
        </div>

        <div class="ramp-container">
          <h3 class="ramp-title">Complimentary Palette</h3>
          <div class="swatch-grid">
            {#each [50, 100, 200, 500, 700] as step}
              <button
                class="swatch"
                style="background-color: var(--palette-comp-{step})"
                on:click={() => copy(`var(--palette-comp-${step})`)}
              >
                <span class="swatch-label">{step}</span>
              </button>
            {/each}
          </div>
        </div>

        <div class="ramp-container">
          <h3 class="ramp-title">Stone (Tinted Neutral)</h3>
          <div class="swatch-grid">
            {#each steps as step}
              <button
                class="swatch"
                style="background-color: var(--palette-stone-{step})"
                on:click={() => copy(`var(--palette-stone-${step})`)}
              >
                <span class="swatch-label">{step}</span>
              </button>
            {/each}
          </div>
        </div>
      </div>
    </section>

    <section class="section">
      <h2 class="section-title">
        <span class="indicator comp"></span> Semantics & Surfaces
      </h2>

      <div class="cards-grid">
        <article class="ui-card panel">
          <div class="card-header">
            <h3>Surface: Panel</h3>
            <span class="badge success">CLEAN</span>
          </div>
          <p>
            This uses <code>--bg-panel</code>. It represents the standard
            content container.
          </p>
          <div class="card-actions">
            <button class="btn btn-primary">Primary</button>
            <button class="btn btn-text">Cancel</button>
          </div>
        </article>

        <article class="ui-card selected">
          <div class="selected-bg-icon">
            <svg
              viewBox="0 0 24 24"
              width="100"
              height="100"
              fill="currentColor"
            >
              <path
                d="M20.24 12.24a6 6 0 0 0-8.49-8.49L5 10.5V19h8.5zM16 8L5 19l-2 2 2 2 2-2 11-11z"
              />
            </svg>
          </div>
          <div class="card-header relative">
            <h3>Surface: Selected</h3>
            <div class="check-circle">✓</div>
          </div>
          <p class="relative">
            This uses <code>--bg-selected</code> and
            <code>--border-active</code>.
          </p>
          <div class="price-tag relative">£350.00</div>
        </article>

        <div class="layer-demo">
          <div class="layer back">
            <span>Layer: Back</span>
          </div>
          <div class="layer middle">
            <span>Layer: Middle</span>
          </div>
          <div class="layer front">
            <span>Layer: Front</span>
            <button class="btn btn-secondary">Comp Action</button>
          </div>
        </div>
      </div>
    </section>

    <section class="section">
      <h2 class="section-title">
        <span class="indicator danger"></span> Intents (Status)
      </h2>
      <div class="intents-grid">
        <div class="intent-box success">
          <div class="dot"></div>
          <div class="intent-text">
            <span class="intent-label">Success</span>
            <span class="intent-sub">Paid / Clean</span>
          </div>
        </div>

        <div class="intent-box warning">
          <div class="dot"></div>
          <div class="intent-text">
            <span class="intent-label">Warning</span>
            <span class="intent-sub">Pending / Inspect</span>
          </div>
        </div>

        <div class="intent-box danger">
          <div class="dot"></div>
          <div class="intent-text">
            <span class="intent-label">Danger</span>
            <span class="intent-sub">Overdue / Dirty</span>
          </div>
        </div>

        <div class="intent-box info">
          <div class="dot"></div>
          <div class="intent-text">
            <span class="intent-label">Info</span>
            <span class="intent-sub">Notes / VIP</span>
          </div>
        </div>
      </div>
    </section>
  </main>
</div>

<style>
  /* =========================================
     SCOPED CSS (The "Pro" Way)
     ========================================= */

  /* Page Container */
  .ds-page {
    min-height: 100vh;
    height: 100vh;
    background-color: var(--bg-back);
    color: var(--text-primary);
    font-family: var(--font-ui, sans-serif);
    transition:
      background-color 0.3s ease,
      color 0.3s ease;
    overflow-y: auto;
    box-sizing: border-box;
  }

  /* Reset for this component */
  .ds-page * {
    box-sizing: border-box;
  }

  /* Wrapper */
  .content-wrapper {
    max-width: 1200px;
    margin: 0 auto;
    padding: 4rem 2rem 10rem;
  }

  /* Typography */
  .page-header {
    margin-bottom: 5rem;
  }

  .page-header h1 {
    font-size: 3rem;
    font-weight: 900;
    letter-spacing: -0.05em;
    margin: 0 0 1rem;
  }

  .page-header p {
    font-size: 1.25rem;
    color: var(--text-secondary);
    max-width: 600px;
    line-height: 1.6;
  }

  .highlight {
    font-family: var(--font-mono, monospace);
    color: var(--palette-brand-500);
    font-weight: bold;
  }

  /* Sections */
  .section {
    margin-bottom: 6rem;
  }

  .section-title {
    font-size: 1.5rem;
    font-weight: 700;
    margin-bottom: 2rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .indicator {
    width: 2rem;
    height: 0.25rem;
    border-radius: 99px;
  }
  .indicator.brand {
    background-color: var(--palette-brand-500);
  }
  .indicator.comp {
    background-color: var(--palette-comp-500);
  }
  .indicator.danger {
    background-color: var(--palette-danger);
  }

  /* =========================================
     PALETTE GRIDS (Primitives)
     ========================================= */
  .grid-layout {
    display: grid;
    gap: 2rem;
  }

  .ramp-title {
    font-family: var(--font-mono, monospace);
    font-size: 0.75rem;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin-bottom: 0.75rem;
    letter-spacing: 0.05em;
  }

  .swatch-grid {
    display: grid;
    grid-template-columns: repeat(10, 1fr); /* The 10-step scale */
    gap: 0.5rem;
    height: 6rem;
  }

  .swatch {
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding: 0.5rem;
    transition: transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .swatch:hover {
    transform: scale(1.1);
    z-index: 2;
  }

  .swatch-label {
    font-family: var(--font-mono, monospace);
    font-size: 0.65rem;
    font-weight: bold;
    opacity: 0.6;
    mix-blend-mode: overlay; /* Allows text to be readable on dark & light */
  }

  /* =========================================
     UI CARDS (Semantics)
     ========================================= */
  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
  }

  .ui-card {
    padding: 2rem;
    border-radius: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    transition:
      transform 0.2s ease,
      box-shadow 0.2s ease;
  }

  /* Standard Panel */
  .ui-card.panel {
    background-color: var(--bg-panel);
    border: 1px solid var(--border-base);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  /* Selected State */
  .ui-card.selected {
    background-color: var(--bg-selected);
    border: 2px solid var(--border-active); /* Stronger border */
    box-shadow: 0 4px 12px -2px rgba(0, 0, 0, 0.1);
    position: relative;
    overflow: hidden;
  }

  .selected-bg-icon {
    position: absolute;
    top: -1rem;
    right: -1rem;
    opacity: 0.1;
    transform: rotate(-12deg) scale(1.5);
    color: var(--palette-brand-500);
    pointer-events: none;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .card-header h3 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 700;
  }

  .relative {
    position: relative;
    z-index: 10;
  }

  .check-circle {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    background-color: var(--palette-brand-500);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: bold;
  }

  .price-tag {
    margin-top: auto;
    font-family: var(--font-mono);
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-brand);
  }

  /* Badges & Buttons */
  .badge {
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    font-weight: 800;
    text-transform: uppercase;
    border: 1px solid transparent;
  }

  .badge.success {
    background-color: var(--color-success-bg);
    color: var(--color-success-fg);
    border-color: rgba(0, 0, 0, 0.1);
  }

  .card-actions {
    display: flex;
    gap: 0.75rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border-dim);
  }

  .btn {
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    font-weight: 700;
    border: none;
    cursor: pointer;
    transition: filter 0.2s;
  }

  .btn:hover {
    filter: brightness(1.1);
  }

  .btn-primary {
    background-color: var(--btn-primary-bg);
    color: var(--btn-primary-fg);
  }

  .btn-secondary {
    background-color: var(--btn-secondary-bg);
    color: var(--btn-secondary-fg);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .btn-text {
    background: none;
    color: var(--text-primary);
  }
  .btn-text:hover {
    background-color: var(--bg-subtle);
  }

  /* =========================================
     LAYERING DEMO
     ========================================= */
  .layer-demo {
    position: relative;
    height: 16rem;
    /* No border or bg here, simpler */
  }

  .layer {
    position: absolute;
    border-radius: 0.75rem;
    padding: 1rem;
    display: flex;
    align-items: flex-start;
    justify-content: flex-end;
  }

  .layer span {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  /* Back Layer */
  .layer.back {
    inset: 0;
    background-color: var(--bg-back);
    border: 1px solid var(--border-dim);
    z-index: 1;
  }

  /* Middle Layer */
  .layer.middle {
    inset: 1.5rem;
    top: 3rem;
    background-color: var(--bg-middle);
    border: 1px solid var(--border-base);
    box-shadow: var(--shadow-md);
    z-index: 2;
  }

  /* Front Layer */
  .layer.front {
    inset: 3rem;
    top: 6rem;
    background-color: var(--bg-front);
    border: 1px solid var(--border-base);
    box-shadow: var(--shadow-lg);
    z-index: 3;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    gap: 1rem;
  }

  /* =========================================
     INTENTS (Status)
     ========================================= */
  .intents-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
  }

  .intent-box {
    padding: 1rem;
    border-radius: 0.75rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    border: 1px solid transparent;
  }

  .dot {
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 50%;
  }

  .intent-text {
    display: flex;
    flex-direction: column;
  }

  .intent-label {
    font-weight: 700;
    font-size: 0.875rem;
  }

  .intent-sub {
    font-size: 0.75rem;
    opacity: 0.8;
  }

  /* Intent Variats - Using CSS Var Mapping */
  .intent-box.success {
    background-color: var(--color-success-bg);
    border-color: rgba(0, 0, 0, 0.05);
  }
  .intent-box.success .dot {
    background-color: var(--color-success-fg);
  }
  .intent-box.success .intent-text {
    color: var(--color-success-fg);
  }

  .intent-box.warning {
    background-color: var(--color-warning-bg);
    border-color: rgba(0, 0, 0, 0.05);
  }
  .intent-box.warning .dot {
    background-color: var(--color-warning-fg);
  }
  .intent-box.warning .intent-text {
    color: var(--color-warning-fg);
  }

  .intent-box.danger {
    background-color: var(--color-danger-bg);
    border-color: rgba(0, 0, 0, 0.05);
  }
  .intent-box.danger .dot {
    background-color: var(--color-danger-fg);
  }
  .intent-box.danger .intent-text {
    color: var(--color-danger-fg);
  }

  .intent-box.info {
    background-color: var(--bg-selected); /* using selected as info bg */
    border-color: var(--border-dim);
  }
  .intent-box.info .dot {
    background-color: var(--palette-brand-500);
  }
  .intent-box.info .intent-text {
    color: var(--palette-brand-700);
  }

  /* =========================================
     FLOATING CONTROLS
     ========================================= */
  .controls {
    position: fixed;
    bottom: 2rem;
    left: 50%;
    transform: translateX(-50%);
    z-index: 100;

    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 1rem 1.5rem;

    background-color: var(--bg-float);
    color: var(--text-on-float);
    border-radius: 1rem;
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .control-group {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    min-width: 100px;
  }

  .control-group label {
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    opacity: 0.7;
    letter-spacing: 0.05em;
  }

  .control-group input {
    width: 100%;
    cursor: pointer;
    accent-color: var(--palette-brand-500);
  }

  .separator {
    width: 1px;
    height: 2rem;
    background-color: rgba(255, 255, 255, 0.2);
  }

  .theme-toggle {
    background: rgba(255, 255, 255, 0.1);
    border: none;
    color: inherit;
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    font-weight: 600;
    font-size: 0.875rem;
    cursor: pointer;
    transition: background 0.2s;
  }

  .theme-toggle:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  /* Responsive adjustment */
  @media (max-width: 768px) {
    .swatch-grid {
      grid-template-columns: repeat(5, 1fr);
      height: auto;
    }
    .controls {
      width: 90%;
      flex-wrap: wrap;
      justify-content: center;
    }
    .layer-demo {
      height: 20rem;
    }
  }
</style>
