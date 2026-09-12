<script lang="ts">
	interface Props {
		size?: number;
		label?: string;
	}

	let { size = 16, label = 'Loading' }: Props = $props();
</script>

<span class="spinner" style:width="{size}px" style:height="{size}px" role="status">
	<span class="visually-hidden">{label}</span>
</span>

<style>
	.spinner {
		display: inline-block;
		border-radius: 50%;
		border: 2px solid var(--color-border);
		border-top-color: currentColor;
		animation: spin 0.6s linear infinite;
		flex-shrink: 0;
	}

	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
