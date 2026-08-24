<script lang="ts">
	import { goto } from '$app/navigation';

	$effect(() => {
		goto('/tape-chart/reservations');
	});
</script>
