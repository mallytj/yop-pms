<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    variant: "primary" | "secondary";
    onclick?: (e: MouseEvent) => void;
    disabled?: boolean;
    dotted?: boolean;
    popoverTarget?: string;
    size?: "sm" | "md" | "lg";
    children: Snippet;
    fill?: boolean;
    className?: string; 
  }

  let {
    variant,
    onclick,
    disabled,
    dotted,
    popoverTarget,
    size,
    children,
    className,
    fill = false,
  }: Props = $props();
</script>

<button
  class={`${variant} ${className}`}
  data-size={size}
  {onclick}
  disabled={disabled ? disabled : undefined}
  popovertarget={popoverTarget}
  data-dotted={dotted}
  type="button"
  data-fill={fill}
>
  {@render children()}
</button>

<style>
  button {
    padding: var(--padding-btn);
    border: var(--border-width-thin) solid var(--border-base);
    border-radius: var(--radius-md);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
    letter-spacing: var(--letter-spacing-wide);
    font-size: var(--font-size-xs);
    background-color: var(--color-light);
    transition: var(--transition-fast) allow-discrete;
    pointer-events: all;
    display: flex;
    gap: var(--gap-sm);
  }

  button[data-fill="true"] {
    width: 100%;
    height: 100%;
  }

  button[data-size="sm"] {
    padding: var(--padding-btn-sm);
    font-size: var(--font-size-2xs);
  }

  button[data-dotted="true"] {
    border-style: dotted;
  }

  button:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }

  button.primary {
    background-color: var(--btn-primary-bg);
    color: var(--btn-primary-fg);
    border-color: var(--border-active);
  }

  button.primary:hover {
    background-color: var(--btn-primary-hover-bg);
    color: var(--btn-primary-hover-fg);
  }

  button.secondary {
    background-color: var(--btn-secondary-bg);
    color: var(--btn-secondary-fg);
  }

  button.secondary:hover {
    background-color: var(--btn-secondary-hover-bg);
    color: var(--btn-secondary-hover-fg);
  }

  button.secondary:active:not([disabled]) {
    background-color: var(--btn-secondary-active-bg);
    color: var(--btn-secondary-active-fg);
  }
</style>
