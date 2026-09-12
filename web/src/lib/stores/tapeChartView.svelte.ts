// Global reactive signal for cross-component tape-chart view requests.
// TopBar's "Today" button lives outside TapeChartGrid's component tree (it's
// rendered from the layout, the grid from the page), so there's no direct
// prop/binding path between them — this counter is the shared channel.
// NOTE: must mutate properties, not reassign `tapeChartView` itself
// (Svelte 5 forbids export of reassignable $state from modules).
export const tapeChartView = $state({
	scrollToTodayRequestId: 0,
	requestScrollToToday() {
		this.scrollToTodayRequestId += 1;
	}
});
