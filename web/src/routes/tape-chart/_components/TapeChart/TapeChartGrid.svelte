<script lang="ts">
	import type { NormalizedTapeChartData } from './_utils/normalizeTapeData.js';
	import TapeChartEmptyState from './_components/TapeChartEmptyState.svelte';
	import TapeChartSkeletonBody from './_components/TapeChartSkeletonBody.svelte';
	import TapeChartLegend from './_components/TapeChartLegend.svelte';
	import RoomRow from './_components/RoomRow.svelte';
	import RoomTypeRow from './_components/RoomTypeRow.svelte';
	import TapeChartHeaderRow from './_components/TapeChartHeaderRow.svelte';
	import type { ReservationCellData } from './_components/RoomAvailabilityCell.svelte';
	import { createVirtualizer } from '@tanstack/svelte-virtual';
	import { addDays, buildDateRange, daysBetween, localDateKey, toDateKey } from '$helpers/dates.js';
	import { nextEdgeLoadDirection, initialScrollLeftForToday } from './_utils/scrollEdge.js';
	import { createLoadGate } from './_utils/loadGate.js';
	import { tapeChartView } from '$stores/tapeChartView.svelte';
	import { untrack } from 'svelte';

	interface Props {
		data: NormalizedTapeChartData;
		loading?: boolean;
		/** Parent fetches [from, to] inclusive and merges. */
		onLoadMore?: (from: string, to: string) => Promise<void> | void;
		/** Defaults to the real current date; inject a fixed value in tests. */
		today?: string;
	}

	let { data, loading = false, onLoadMore, today: todayProp }: Props = $props();

	const CELL_WIDTH_PX = 96;
	const ROOM_COLUMN_WIDTH_PX = 160;
	const ROW_HEIGHT_PX = 42;
	const HEADER_HEIGHT_PX = 52;
	const BATCH_DAY_COUNT = 30;
	// Scrolling holds the edge check satisfied for as long as the operator
	// keeps the scrollbar thumb at the boundary, so even a successful load
	// must wait out this cooldown before the gate reopens — otherwise it
	// refires the next batch instantly and a single scroll-to-edge triggers a
	// burst of loads instead of one.
	const LOAD_COOLDOWN_MS = 2000;
	/** Trigger the next fetch once the scroll edge gets this close to loaded data's end. */
	// A fast scroll can cover a small buffer before a fetch resolves, hitting
	// the edge of the loaded (and thus scrollable) range before more days
	// arrive — the scrollbar physically can't move further until `dates`
	// grows. A wider buffer gives the fetch more time to land first.
	const EDGE_THRESHOLD_COLUMNS = 20;

	let today = $derived(todayProp ?? localDateKey());
	let roomColumnWidth = $state(ROOM_COLUMN_WIDTH_PX);
	let rangeFrom = $derived(data.from);
	let rangeTo = $derived(data.to);
	let dates = $derived(buildDateRange(rangeFrom, rangeTo));

	interface CellData {
		status: string;
		reservation?: ReservationCellData;
	}

	interface MaintenanceCellData {
		id: string;
		blockType: string;
	}

	let inventoryByRoomAndDate: Map<string, Map<string, CellData>> = $derived.by(() => {
		const cells = new Map<string, Map<string, CellData>>();
		for (const day of data.inventory) {
			const roomId = day.room_id ?? '';
			const calendarDate = toDateKey(day.calendar_date);

			if (!roomId || !calendarDate) continue;
			if (!cells.has(roomId)) cells.set(roomId, new Map());

			cells.get(roomId)!.set(calendarDate, { status: day.status ?? '' });
		}

		for (const reservation of data.reservations) {
			const roomId = reservation.room_id ?? '';
			const startDate = toDateKey(reservation.from);
			const endDate = toDateKey(reservation.to);
			if (!roomId || !startDate || !endDate) continue;

			const cellReservation: ReservationCellData = {
				code: reservation.code ?? '',
				guestName: reservation.guest_name ?? reservation.code ?? '',
				status: reservation.status ?? '',
				accentIndex: reservation.accent_index,
				...(reservation.price_pence !== undefined ? { pricePence: reservation.price_pence } : {})
			};

			if (!cells.has(roomId)) cells.set(roomId, new Map());
			const roomCells = cells.get(roomId)!;
			const nights = daysBetween(startDate, endDate);

			for (let offset = 0; offset < nights; offset += 1) {
				const date = addDays(startDate, offset);
				const existingStatus = roomCells.get(date)?.status ?? '';
				roomCells.set(date, { status: existingStatus, reservation: cellReservation });
			}
		}
		return cells;
	});

	let maintenanceByRoomAndDate: Map<string, Map<string, MaintenanceCellData>> = $derived.by(() => {
		const cells = new Map<string, Map<string, MaintenanceCellData>>();
		for (const block of data.maintenanceBlocks) {
			const id = block.id ?? '';
			const roomId = block.room_id ?? '';
			const startDate = toDateKey(block.start);
			const endDate = toDateKey(block.end);
			if (!id || !roomId || !startDate || !endDate) continue;

			const cellData: MaintenanceCellData = { id, blockType: block.reason ?? '' };
			if (!cells.has(roomId)) cells.set(roomId, new Map());

			const roomCells = cells.get(roomId)!;
			const nights = daysBetween(startDate, endDate);

			for (let offset = 0; offset < nights; offset += 1) {
				roomCells.set(addDays(startDate, offset), cellData);
			}
		}
		return cells;
	});

	let roomTypeGroups: { id: string; name: string; rooms: { id: string; name: string }[] }[] =
		$derived(
			data.roomTypes.map((roomType) => ({
				id: roomType.id ?? '',
				name: roomType.name ?? '',
				rooms: (roomType.rooms ?? []).map((room) => ({
					id: room.id ?? '',
					name: room.name ?? ''
				}))
			}))
		);

	function cellData(roomId: string, date: string): CellData {
		return inventoryByRoomAndDate.get(roomId)?.get(date) ?? { status: '' };
	}

	function maintenanceCellData(roomId: string, date: string): MaintenanceCellData | undefined {
		return maintenanceByRoomAndDate.get(roomId)?.get(date);
	}

	const loadGate = createLoadGate(LOAD_COOLDOWN_MS);

	function loadMore(direction: 'past' | 'future') {
		const edge = direction === 'past' ? rangeFrom : rangeTo;
		if (!edge || !onLoadMore) return;

		const from = direction === 'past' ? addDays(edge, -BATCH_DAY_COUNT) : addDays(edge, 1);
		const to = direction === 'past' ? addDays(edge, -1) : addDays(edge, BATCH_DAY_COUNT);

		Promise.resolve(onLoadMore(from, to)).then(
			() => loadGate.settle(true),
			() => loadGate.settle(false)
		);
	}

	type GridRow =
		| { kind: 'roomType'; id: string; name: string }
		| { kind: 'room'; id: string; room: { id: string; name: string } };

	let gridRows = $derived.by<GridRow[]>(() =>
		roomTypeGroups.flatMap((roomType) => [
			{ kind: 'roomType' as const, id: `type:${roomType.id}`, name: roomType.name },
			...roomType.rooms.map((room) => ({
				kind: 'room' as const,
				id: `room:${room.id}`,
				room
			}))
		])
	);

	let scrollElement: HTMLDivElement | null = $state(null);
	const rowVirtualizer = createVirtualizer<HTMLDivElement, HTMLDivElement>({
		getScrollElement: () => scrollElement,
		count: 0,
		estimateSize: () => ROW_HEIGHT_PX,
		overscan: 8
	});

	// @tanstack/svelte-virtual's setOptions() unconditionally re-publishes the
	// virtualizer store (see its `createVirtualizerBase`), so reading `$rowVirtualizer`
	// (auto-subscribe) to call it inside this same effect would make the effect
	// depend on the value it just wrote — an infinite update loop. `untrack`
	// keeps the write out of the effect's dependency set; only `gridRows.length`
	// (the thing that should actually retrigger this) is tracked.
	$effect(() => {
		const count = gridRows.length;
		untrack(() => $rowVirtualizer.setOptions({ count }));
	});

	let virtualRows = $derived($rowVirtualizer.getVirtualItems());
	let virtualHeight = $derived($rowVirtualizer.getTotalSize());
	const columnVirtualizer = createVirtualizer<HTMLDivElement, HTMLDivElement>({
		getScrollElement: () => scrollElement,
		count: 0,
		estimateSize: () => CELL_WIDTH_PX,
		horizontal: true,
		overscan: 4
	});

	$effect(() => {
		const count = dates.length;
		untrack(() => $columnVirtualizer.setOptions({ count }));
	});

	let virtualColumns = $derived($columnVirtualizer.getVirtualItems());
	let dateOffset = $derived(virtualColumns[0]?.index ?? 0);
	let visibleDates = $derived(virtualColumns.map((column) => dates[column.index]));

	// Bidirectional infinite scroll: fetch the next batch once the scrolled-to
	// edge gets within EDGE_THRESHOLD_COLUMNS of loaded data's start/end.
	// Gated on an actual scroll event — without it, the initial viewport
	// (column 0 visible) trivially satisfies the "near the start" edge check
	// and fires a fetch (repeatedly, since the prepend keeps landing back
	// near the edge) before the operator has scrolled at all.
	let hasScrolledHorizontally = false;

	// Set right before an imperative `scrollElement.scrollLeft = ...` write
	// (e.g. the initial today-positioning below) so the `scroll` event that
	// write dispatches isn't mistaken for operator input — otherwise it
	// flips `hasScrolledHorizontally` and can trip the edge check before the
	// operator has touched the scrollbar, same failure mode as above.
	let suppressNextScrollEvent = false;

	function checkEdgeAndMaybeLoad() {
		if (!onLoadMore) return;
		const direction = nextEdgeLoadDirection({
			hasScrolledHorizontally,
			loading,
			pendingLoadDirection: loadGate.pending,
			columns: $columnVirtualizer.getVirtualItems(),
			totalDates: dates.length,
			edgeThreshold: EDGE_THRESHOLD_COLUMNS
		});
		if (!direction || !loadGate.start(direction)) return;

		loadMore(direction);
	}

	$effect(() => {
		const element = scrollElement;
		if (!element) return;

		const onScroll = () => {
			if (suppressNextScrollEvent) {
				suppressNextScrollEvent = false;
				return;
			}
			hasScrolledHorizontally = true;
		};
		// `scroll` only fires when scrollLeft actually changes — once it's
		// clamped at the track's end, continuing to scroll in that direction
		// produces no further `scroll` events, so the reactive check below
		// (keyed on `virtualColumns` changing) never re-runs and the operator
		// gets stuck at the wall until they scroll back and forth to nudge it.
		// `wheel` fires on every gesture regardless of whether the position
		// moved, so it can trigger the check directly at that clamped edge.
		const onWheel = () => {
			hasScrolledHorizontally = true;
			checkEdgeAndMaybeLoad();
		};
		element.addEventListener('scroll', onScroll, { passive: true });
		element.addEventListener('wheel', onWheel, { passive: true });

		return () => {
			element.removeEventListener('scroll', onScroll);
			element.removeEventListener('wheel', onWheel);
		};
	});

	// Every dependency this decision needs is read unconditionally, before
	// the `onLoadMore`/gate short-circuit — a guard clause that returns before
	// reading `virtualColumns` would otherwise silently unsubscribe this
	// effect from it (Svelte only tracks reads that actually execute), so it
	// would only ever fire once, on mount.
	$effect(() => {
		// Forces the tracked reads `checkEdgeAndMaybeLoad` needs — it re-reads
		// them itself (see its own comment for why), these are only here so
		// Svelte re-runs this effect when any of them changes.
		void virtualColumns;
		void dates.length;
		void loading;
		checkEdgeAndMaybeLoad();
	});

	// Landing at scrollLeft 0 pins the scrollbar thumb to its track minimum —
	// physically impossible to drag further left — even though the initial
	// fetch already includes a lookback buffer of past days (see
	// DEFAULT_LOOKBACK_DAYS in $lib/api/tape-chart.ts). Positioning at `today`
	// instead puts that buffer to the left of the viewport as real scroll
	// headroom. Runs once: a plain variable (not $state) so writing it here
	// doesn't retrigger this same effect.
	let hasSetInitialScrollLeft = false;
	$effect(() => {
		if (hasSetInitialScrollLeft || !scrollElement || dates.length === 0) return;
		hasSetInitialScrollLeft = true;

		const initialScrollLeft = initialScrollLeftForToday(dates, today, CELL_WIDTH_PX);
		if (initialScrollLeft === 0) return;

		suppressNextScrollEvent = true;
		scrollElement.scrollLeft = initialScrollLeft;
	});

	// TopBar's "Today" button lives outside this component's tree, so it
	// signals here via a shared counter (see tapeChartView.svelte.ts) rather
	// than a prop. Baselined at mount so a request from a *previous* grid
	// instance (e.g. the operator clicked Today, then navigated elsewhere)
	// doesn't force an unwanted scroll the moment a new grid mounts.
	let lastHandledScrollToTodayRequestId = tapeChartView.scrollToTodayRequestId;
	$effect(() => {
		const requestId = tapeChartView.scrollToTodayRequestId;
		if (requestId === lastHandledScrollToTodayRequestId || !scrollElement || dates.length === 0) {
			return;
		}
		lastHandledScrollToTodayRequestId = requestId;
		suppressNextScrollEvent = true;
		scrollElement.scrollLeft = initialScrollLeftForToday(dates, today, CELL_WIDTH_PX);
	});

	// Prepending days shifts every already-visible column to the right;
	// compensate scrollLeft so the dates the operator was looking at don't
	// jump. Plain (non-$state) variable: it only needs to persist across
	// effect runs, not participate in reactivity — tracking it as $state
	// would make this effect depend on a value it also writes.
	let previousRangeFrom = '';
	$effect(() => {
		const currentRangeFrom = rangeFrom;
		if (
			previousRangeFrom &&
			currentRangeFrom &&
			currentRangeFrom < previousRangeFrom &&
			scrollElement
		) {
			const prependedDays = daysBetween(currentRangeFrom, previousRangeFrom);
			scrollElement.scrollLeft += prependedDays * CELL_WIDTH_PX;
		}
		previousRangeFrom = currentRangeFrom;
	});

	let gridWidth = $derived(roomColumnWidth + dates.length * CELL_WIDTH_PX);
	// Dates come from the requested range and are known before room/reservation
	// data resolves, so the header can render immediately — only the body
	// (which needs room data) waits, as a skeleton, while `loading` is true.
	let hasNoDates = $derived(dates.length === 0);
	let hasNoRooms = $derived(roomTypeGroups.length === 0);
	let emptyHint = $derived(
		hasNoDates
			? 'Choose a start date to see availability.'
			: 'Add a room to this property to see its availability.'
	);
</script>

<div
	class="tape-chart"
	style="--cell-w: {CELL_WIDTH_PX}px; --room-w: {roomColumnWidth}px; --row-h: {ROW_HEIGHT_PX}px; --header-h: {HEADER_HEIGHT_PX}px;"
>
	{#if hasNoDates}
		<TapeChartEmptyState title="No dates in range" hint={emptyHint} />
	{:else}
		<div class="tape-chart-scroll" bind:this={scrollElement}>
			<div
				class="tape-chart-table"
				role="grid"
				aria-label="Room availability"
				style="width: {gridWidth}px;"
			>
				<TapeChartHeaderRow
					dates={visibleDates}
					{dateOffset}
					{today}
					{roomColumnWidth}
					onRoomColumnResize={(nextWidth) => (roomColumnWidth = nextWidth)}
				/>

				{#if hasNoRooms && loading}
					<TapeChartSkeletonBody columnCount={visibleDates.length} />
				{:else if hasNoRooms}
					<TapeChartEmptyState title="No rooms to display" hint={emptyHint} />
				{:else}
					<div class="tape-chart-virtual-body" style="height: {virtualHeight}px;">
						{#each virtualRows as virtualRow (virtualRow.key)}
							{@const row = gridRows[virtualRow.index]}
							{#if row.kind === 'roomType'}
								<div
									class="tape-chart-virtual-row"
									style="transform: translateY({virtualRow.start}px);"
								>
									<RoomTypeRow
										name={row.name}
										dates={visibleDates}
										{dateOffset}
										{roomColumnWidth}
										onRoomColumnResize={(nextWidth) => (roomColumnWidth = nextWidth)}
									/>
								</div>
							{:else}
								<div
									class="tape-chart-virtual-row"
									style="transform: translateY({virtualRow.start}px);"
								>
									<RoomRow
										room={row.room}
										dates={visibleDates}
										{dateOffset}
										{today}
										getCell={cellData}
										getMaintenanceCell={maintenanceCellData}
										{roomColumnWidth}
										onRoomColumnResize={(nextWidth) => (roomColumnWidth = nextWidth)}
									/>
								</div>
							{/if}
						{/each}
					</div>
				{/if}
			</div>
		</div>

		<TapeChartLegend {rangeFrom} {rangeTo} loading={loading && !hasNoRooms} />
	{/if}
</div>

<style>
	.tape-chart {
		position: relative;
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: var(--color-canvas);
		color: var(--color-text);
		font-family: var(--font-sans);
		font-size: var(--font-size-sm);
		line-height: 1.4;
		-webkit-font-smoothing: antialiased;
		box-sizing: border-box;
	}

	.tape-chart *,
	.tape-chart *::before,
	.tape-chart *::after {
		box-sizing: border-box;
	}

	.tape-chart-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		overscroll-behavior: contain;
	}

	.tape-chart-scroll::-webkit-scrollbar {
		width: 10px;
		height: 10px;
	}

	.tape-chart-scroll::-webkit-scrollbar-thumb {
		background: var(--color-scrollbar-thumb);
		border-radius: 8px;
		border: 2px solid transparent;
		background-clip: content-box;
	}

	.tape-chart-scroll::-webkit-scrollbar-thumb:hover {
		background-color: var(--color-scrollbar-thumb-hover);
	}

	.tape-chart-scroll::-webkit-scrollbar-track,
	.tape-chart-scroll::-webkit-scrollbar-corner {
		background: transparent;
	}

	.tape-chart-table {
		display: flex;
		flex-direction: column;
	}

	.tape-chart-virtual-body {
		position: relative;
	}

	.tape-chart-virtual-row {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: var(--row-h);
	}
</style>
