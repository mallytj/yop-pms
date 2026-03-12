<script lang="ts">
  import { untrack } from "svelte";

  type InputType = "number" | "text" | "date";
  type Size = "xs" | "sm" | "md" | "lg";
  type InputMode =
    | "text"
    | "decimal"
    | "numeric"
    | "none"
    | "search"
    | "tel"
    | "url"
    | "email"
    | null
    | undefined;

  interface Props {
    inputType: InputType;
    size?: Size;
    suffix?: string;
    prefix?: string;
    placeholder?: string;
    value: number | string | undefined;
    name?: string;
    color?: "subtle" | "base";
    dp?: number;
    onblur?: (e: FocusEvent) => void;
    fontSize?: string;
  }

  let {
    inputType,
    suffix,
    prefix,
    placeholder,
    value = $bindable(),
    name = `${inputType}-input`,
    color,
    dp,
    onblur,
    fontSize = "var(--font-size-sm)",
    size = "sm",
  }: Props = $props();

  // Internal state for the input value
  let internalText = $state(
    value !== undefined && value !== null ? String(value) : "",
  );

  let internallyDriven = false;

  const inputMode = $derived.by<InputMode>(() => {
    if (inputType === "number") {
      return dp && dp > 0 ? "decimal" : "numeric";
    }
    if (inputType === "date") return "none";

    return "text";
  });

  const resolvedType = $derived(inputType === "number" ? "text" : inputType);

  function handleInput(e: Event) {
    const raw = (e.target as HTMLInputElement).value;
    internalText = raw;

    internallyDriven = true;
    if (inputType === "number") {
      const parsed = dp && dp > 0 ? parseFloat(raw) : parseInt(raw, 10);
      value = isNaN(parsed) ? undefined : parsed;
    } else {
      value = raw;
    }
    internallyDriven = false;
  }

  function handleBlur(e: FocusEvent) {
    // Normalise display on blur for numbers
    if (inputType === "number" && typeof value === "number") {
      const formatted = dp && dp > 0 ? value.toFixed(dp) : String(value);
      internalText = formatted && !isNaN(Number(formatted)) ? formatted : "";
    }
    onblur?.(e);
  }

  function parse(raw: string): number | undefined {
    if (raw === "" || raw === "-") return undefined;
    const parsed = dp && dp > 0 ? parseFloat(raw) : parseInt(raw, 10);
    return isNaN(parsed) ? undefined : parsed;
  }

  // Sync internal state with value
  $effect(() => {
    const incoming = value;
    const current = untrack(() => internalText);
    const parsedCurrent = parse(current);

    if (parsedCurrent === incoming) return;

    internalText =
      incoming !== undefined && incoming !== null ? String(incoming) : "";
  });
</script>

<div
  class="input-group"
  class:subtle={color === "subtle"}
  style="--font-size-override: {fontSize}"
  data-size={size}
>
  {#if prefix}
    <span class="prefix">{prefix}</span>
  {/if}
  <input
    inputmode={inputMode}
    type={resolvedType}
    oninput={handleInput}
    value={internalText}
    {placeholder}
    step={dp! > 0 ? 1 / Math.pow(10, dp!) : "1"}
    {name}
    class="native-input"
    onblur={handleBlur}
  />

  {#if suffix}
    <span class="suffix">{suffix}</span>
  {/if}
</div>

<style>
  .input-group {
    --font-size: var(--font-size-override, var(--font-size-sm));
    --padding: var(--padding-btn-sm);
    --gap: var(--gap-sm);
  }

  .input-group[data-size="xs"] {
    --font-size: var(--font-size-override, var(--font-size-xs));
    --padding: var(--padding-btn-xs);
    --gap: var(--gap-xs);
  }

  .input-group[data-size="sm"] {
    --font-size: var(--font-size-override, var(--font-size-sm));
    --padding: var(--padding-btn-sm);
    --gap: var(--gap-sm);
  }

  .input-group[data-size="md"] {
    --font-size: var(--font-size-override, var(--font-size-md));
    --padding: var(--padding-btn-md);
    --gap: var(--gap-md);
  }

  .input-group[data-size="lg"] {
    --font-size: var(--font-size-override, var(--font-size-lg));
    --padding: var(--padding-btn-lg);
    --gap: var(--gap-lg);
  }

  .input-group {
    display: flex;
    align-items: center;
    gap: var(--gap);
    min-width: 100%;
    width: 100%;
    padding: var(--padding);
    border-radius: var(--radius-md);
    background-color: var(--bg-panel);
    border: var(--border-width-thin) solid var(--border-base);
    color: var(--text-primary);

    transition: var(--transition-fast);
    cursor: text;
  }

  .input-group.subtle {
    background-color: var(--bg-subtle);
    color: var(--fg-subtle);
  }

  .input-group:focus-within {
    border-color: var(--border-active);
    box-shadow: 0 0 0 3px var(--bg-selected);
  }

  .native-input {
    flex: 1 1 0;
    field-sizing: content;
    border: none;
    background: transparent;
    outline: none;
    min-width: 0;
    width: 100%;
    padding: 0;
    margin: 0;
    font-family: inherit;
    font-size: var(--font-size);
    font-weight: var(--font-weight-medium);
    color: var(--fg-subtle);
    font-family: var(--font-data);

    -moz-appearance: textfield;
    appearance: textfield;
  }

  .native-input::-webkit-outer-spin-button,
  .native-input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }

  input[type="date"]::-webkit-calendar-picker-indicator {
    display: none;
    appearance: none;
    margin: 0;
    -webkit-appearance: none;
  }

  .prefix,
  .suffix {
    flex-shrink: 0;
    font-family: var(--font-data);
    color: var(--text-secondary);
    font-weight: var(--font-weight-semibold);
    user-select: none;
    font-size: var(--font-size);
  }

  .suffix {
    font-size: calc(var(--font-size) * 0.75);
  }
</style>
