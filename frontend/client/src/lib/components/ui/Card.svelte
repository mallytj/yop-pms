<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    size?: "sm" | "md" | "lg" | "xl";
    children: Snippet;
  }
  let { size = "md", children }: Props = $props();
</script>

<div class="card-container">
  {@render children()}
</div>

<style>
</style>
