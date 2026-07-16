<script lang="ts">
	import { goto } from '$app/navigation';

	$effect(() => {
		goto('/planner/reservations');
	});
</script>
