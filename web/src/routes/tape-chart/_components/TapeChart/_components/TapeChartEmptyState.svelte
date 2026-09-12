<script lang="ts">
	interface Props {
		title: string;
		hint: string;
	}

	let { title, hint }: Props = $props();
</script>

<div class="empty-state" role="status">
	<h3 class="empty-title">{title}</h3>
	<p class="empty-hint">{hint}</p>
</div>

<style>
	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--spacing-sm);
		padding: var(--spacing-xl) var(--spacing-lg);
		text-align: center;
	}

	.empty-title {
		margin: 0;
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text);
		text-wrap: balance;
	}

	.empty-hint {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		text-wrap: pretty;
	}
</style>
