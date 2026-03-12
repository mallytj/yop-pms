<script lang="ts">
  import { toasts, dismissToast } from "../stores/toast_store.ts";
  import { fly } from "svelte/transition";
  import { flip } from "svelte/animate";
</script>

<div class="toast-container">
  {#each $toasts as toast (toast.id)}
    <div
      class="toast {toast.type}"
      role="alert"
      animate:flip
      transition:fly={{ y: 20, duration: 300 }}
    >
      <div class="message">{toast.message}</div>

      <button
        class="close-btn"
        on:click={() => dismissToast(toast.id)}
        aria-label="Close notification"
      >
        &times;
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: var(--gap-xl);
    right: var(--gap-xl);
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: var(--gap-sm);
    pointer-events: none;
  }

  .toast {
    pointer-events: auto;
    min-width: 250px;
    padding: var(--padding-btn);
    border-radius: var(--radius-md);
    background: var(--color-bg-dark);
    color: var(--color-light);
    box-shadow: var(--shadow-md);
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
    font-size: var(--font-size-sm);
  }

  /* Variant Colors */
  .toast.error {
    background-color: var(--color-danger); /* Red */
  }
  .toast.success {
    background-color: var(--color-success); /* Green */
  }
  .toast.info {
    background-color: var(--color-warning); /* Blue */
  }

  .close-btn {
    background: transparent;
    border: none;
    color: white;
    font-size: var(--font-size-md);
    margin-left: var(--gap-md);
    cursor: pointer;
    opacity: 0.7;
  }

  .close-btn:hover {
    opacity: 1;
  }
</style>
