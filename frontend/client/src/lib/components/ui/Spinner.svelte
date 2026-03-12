<script lang="ts">
  interface Props {
    size: "sm" | "md" | "lg";
    color?: string;
  }

  let { size, color }: Props = $props();
</script>

<div class="spinner spinner--{size}" style="--spinner-color: {color}">
  <span class="bar"></span>
  <span class="bar"></span>
  <span class="bar"></span>
  <span class="bar"></span>
</div>

<style>
  .spinner {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }

  .spinner--sm .bar {
    width: 2px;
    height: 14px;
  }
  .spinner--md .bar {
    width: 3px;
    height: 22px;
  }
  .spinner--lg .bar {
    width: 4px;
    height: 32px;
  }

  .bar {
    background-color: var(--spinner-color);
    border-radius: var(--radius-sm);
    animation: bounce 1s ease-in-out infinite;
  }

  .bar:nth-child(1) {
    animation-delay: 0s;
  }
  .bar:nth-child(2) {
    animation-delay: 0.1s;
  }
  .bar:nth-child(3) {
    animation-delay: 0.2s;
  }
  .bar:nth-child(4) {
    animation-delay: 0.3s;
  }

  @keyframes bounce {
    0%,
    40%,
    100% {
      transform: scaleY(0.4);
      opacity: 0.4;
    }
    20% {
      transform: scaleY(1);
      opacity: 1;
    }
  }
</style>
