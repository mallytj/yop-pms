<script lang="ts">
  import { getContext } from "svelte";
  import { Button } from ".";
  import Input from "./Input.svelte";

  interface Props {
    id: string;
    max?: number;
    min?: number;
    defaultVal: number;
    onchange?: (v: number) => void;
  }

  let { id, max, min, defaultVal, onchange }: Props = $props();
  let value: number = $state<number>(defaultVal);

  function stepDown() {
    if (value == min) return;
    value--;
    onchange!(value);
  }

  function stepUp() {
    if (value == max) return;
    value++;
    onchange!(value);
  }
</script>

<div class="stepper-container">
  <div class="stepper stepper--down">
    <Button
      size="sm"
      variant="secondary"
      onclick={stepDown}
      disabled={value == min}
      fill={true}>-</Button
    >
  </div>
  <div class="number">
    <Input inputType="number" placeholder={String(defaultVal)} {value}></Input>
  </div>
  <div class="stepper stepper--up">
    <Button
      size="sm"
      variant="secondary"
      onclick={stepUp}
      disabled={value == max}
      className="stepper-btn"
      fill={true}>+</Button
    >
  </div>
</div>

<style>
  .stepper-container {
    display: flex;
    gap: var(--gap-sm);
    place-items: center;
    height: 100%;
    container-type: sidebar / inline-size;
  }

  :global(.stepper-btn) {
    height: 100%;
    font-family: var(--font-data) !important;
    font-weight: var(--font-weight-extrabold) !important;
  }

  .number {
    color: var(--fg-subtle);
    font-size: var(--font-size-sm);
  }
</style>
