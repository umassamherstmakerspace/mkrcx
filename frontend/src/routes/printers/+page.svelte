<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from 'flowbite-svelte';
	import {
		printers as unavailableFleet,
		duration,
		finishTime,
		type Printer,
		type Condition
	} from '$lib/printers/prototype-data';
	import {
		sortFleet,
		remainingMinutes,
		type SortKey,
		type SortDirection
	} from '$lib/printers/fleet-view';
	let printers = unavailableFleet;
	let staffView = false;
	let filter = 'all';
	let disconnected = true;
	let fetchedAt: string | null = null;
	let expandedId: string | null = null;
	let sortKey: SortKey = 'condition';
	let sortDirection: SortDirection = 'asc';
	const conditionText: Record<Condition, string> = {
		working: 'Working',
		limited: 'Limited use',
		out: 'Out of service',
		unknown: 'Status unavailable'
	};
	const columns: { key: SortKey | null; label: string; className: string }[] = [
		{ key: 'name', label: 'Printer', className: 'printer-column' },
		{ key: 'model', label: 'Model', className: 'model-column' },
		{ key: 'condition', label: 'Status', className: 'condition-column' },
		{ key: 'activity', label: 'Activity', className: 'activity-column' },
		{ key: 'remaining', label: 'Est. time left', className: 'time-column' },
		{ key: null, label: 'Notes', className: 'note-column' }
	];
	const filters = [
		{ id: 'all', label: 'All printers' },
		{ id: 'idle', label: 'Idle' },
		{ id: 'printing', label: 'Printing' },
		{ id: 'limited', label: 'Limited use' },
		{ id: 'out', label: 'Out of service' }
	];
	async function refresh() {
		try {
			const response = await fetch('/printers/data');
			if (!response.ok) throw new Error('fleet request failed');
			const fleet = (await response.json()) as {
				audience: 'public' | 'staff';
				stale: boolean;
				fetchedAt: string | null;
				printers: Printer[];
			};
			if (fleet.printers.length !== unavailableFleet.length) throw new Error('incomplete fleet');
			printers = fleet.printers;
			staffView = fleet.audience === 'staff';
			disconnected = fleet.stale;
			fetchedAt = fleet.fetchedAt;
		} catch {
			disconnected = true;
			staffView = false;
			printers = unavailableFleet;
			fetchedAt = null;
		}
	}
	onMount(() => {
		void refresh();
		const timer = window.setInterval(refresh, 15_000);
		return () => window.clearInterval(timer);
	});
	function matches(printer: Printer, selected: string) {
		if (selected === 'all') return true;
		if (selected === 'idle')
			return (
				!disconnected &&
				!printer.stale &&
				printer.activity === 'idle' &&
				['working', 'limited'].includes(printer.condition)
			);
		if (selected === 'printing')
			return !disconnected && !printer.stale && printer.activity === 'printing';
		return printer.condition === selected;
	}
	$: visible = sortFleet(
		printers.filter((printer) => matches(printer, filter)),
		sortKey,
		sortDirection,
		disconnected
	);
	$: if (!staffView) expandedId = null;
	function changeSort(key: SortKey) {
		sortDirection = sortKey === key && sortDirection === 'asc' ? 'desc' : 'asc';
		sortKey = key;
	}
	function resetSort() {
		sortKey = 'condition';
		sortDirection = 'asc';
	}
	function activityLabel(printer: Printer) {
		if (disconnected || printer.stale || printer.activity === 'unknown') return 'Unknown';
		return printer.activity === 'printing'
			? 'Printing'
			: printer.activity === 'paused'
				? 'Paused'
				: 'Idle';
	}
	function conditionIcon(condition: Condition) {
		return { working: '✓', limited: '!', out: '×', unknown: '—' }[condition];
	}
	function updateTime(value: string | null) {
		return value
			? new Date(value).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
			: 'Unavailable';
	}
</script>

<svelte:head>
	<title>3D Printer Fleet Status · UMass Makerspace</title>
	<meta
		name="description"
		content="See printer conditions, public notes and estimated print times at the UMass Makerspace."
	/>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<main class="printer-dashboard">
	<header class="page-heading">
		<div class="title-line">
			<h1>3D Printer Fleet Status</h1>
			<span class="sample-label">{staffView ? 'Staff view' : 'Public view'}</span>
		</div>
	</header>
	{#if disconnected}<div class="connection-banner" role="status">
			<strong>Updates unavailable</strong> Printer condition, activity, and finish estimates cannot be
			confirmed.
		</div>{/if}
	<div class="toolbar">
		<div class="filters" role="group" aria-label="Filter printers">
			{#each filters as item}<Button
					color="none"
					size="sm"
					class={filter === item.id ? 'filter-button selected' : 'filter-button'}
					aria-pressed={filter === item.id}
					on:click={() => (filter = item.id)}
					>{item.label}<span class="filter-count"
						>{printers.filter((printer) => matches(printer, item.id)).length}</span
					></Button
				>{/each}
		</div>
		{#if sortKey !== 'condition' || sortDirection !== 'asc'}<Button
				color="none"
				size="xs"
				class="reset-sort"
				on:click={resetSort}>Restore default order</Button
			>{/if}
	</div>
	<div
		class="fleet-table-scroll"
		role="region"
		aria-label={staffView ? 'Staff printer fleet status' : 'Public printer fleet status'}
	>
		<table class="fleet-table">
			<caption class="visually-hidden"
				>Default order: working, limited use, out of service, then unavailable. Within each status,
				printing and paused come before idle, then K1 Max before K1 and K1C. Select a column heading
				to change sort order. {staffView
					? 'Select a printer name to expand its details.'
					: ''}</caption
			>
			<colgroup
				>{#each columns as column}<col class={column.className} />{/each}</colgroup
			>
			<thead
				><tr
					>{#each columns as column}<th
							scope="col"
							aria-sort={sortKey === column.key
								? sortDirection === 'asc'
									? 'ascending'
									: 'descending'
								: column.key
									? 'none'
									: undefined}
							>{#if column.key}<button
									type="button"
									class="sort-button"
									class:sort-active={sortKey === column.key}
									on:click={() => column.key && changeSort(column.key)}
									aria-label={`Sort by ${column.label}, ${sortKey === column.key && sortDirection === 'asc' ? 'descending' : 'ascending'}`}
									>{column.label}<span class="sort-arrow" aria-hidden="true"
										>{sortKey === column.key ? (sortDirection === 'asc' ? '↑' : '↓') : '↕'}</span
									></button
								>{:else}<span class="static-heading">{column.label}</span>{/if}</th
						>{/each}</tr
				></thead
			>
			<tbody>
				{#each visible as printer (printer.id)}
					{@const minutes = remainingMinutes(printer, disconnected)}
					<tr
						class:row-working={printer.condition === 'working'}
						class:row-limited={printer.condition === 'limited'}
						class:row-out={printer.condition === 'out'}
						class:row-stale={disconnected || printer.stale || printer.condition === 'unknown'}
					>
						<th scope="row">
							{#if staffView}<button
									class="printer-expand"
									type="button"
									aria-expanded={expandedId === printer.id}
									aria-controls={`details-${printer.id}`}
									aria-label={`${expandedId === printer.id ? 'Close' : 'View'} details for ${printer.name}`}
									on:click={() => (expandedId = expandedId === printer.id ? null : printer.id)}
									><span
										class="chevron"
										class:expanded={expandedId === printer.id}
										aria-hidden="true">›</span
									><span class="table-name">{printer.name}</span></button
								>{:else}<span class="table-name">{printer.name}</span>{/if}
						</th>
						<td class="table-model">{printer.model}</td>
						<td
							><span class="condition-badge {printer.condition}"
								><span aria-hidden="true">{conditionIcon(printer.condition)}</span>{conditionText[
									printer.condition
								]}</span
							></td
						>
						<td
							class:table-printing={printer.activity === 'printing' &&
								!disconnected &&
								!printer.stale}>{activityLabel(printer)}</td
						>
						<td class="table-time"
							>{#if minutes !== null}<span title={`Estimated finish ${finishTime(minutes)}`}
									>~{duration(minutes)}</span
								>{:else}<span class="table-muted"
									>{printer.activity === 'printing' && !disconnected && !printer.stale
										? 'Unavailable'
										: '—'}</span
								>{/if}</td
						>
						<td class="table-note">{printer.note ?? (printer.stale ? 'No recent update.' : '')}</td>
					</tr>
					{#if staffView && expandedId === printer.id}
						<tr class="details-row"
							><td colspan="6"
								><section
									id={`details-${printer.id}`}
									aria-label={`${printer.name} details`}
									class="print-details"
								>
									<div class="detail-heading">
										<strong>{printer.name} · Current print</strong><Button
											color="none"
											size="xs"
											class="close-details"
											on:click={() => (expandedId = null)}>Close</Button
										>
									</div>
									{#if disconnected || printer.stale}<p class="detail-empty">
											Current print details are unavailable until the printer reconnects.
										</p>
									{:else if printer.job && ['printing', 'paused'].includes(printer.activity)}
										<dl class="detail-fields">
											<div>
												<dt>Printing for</dt>
												<dd>{printer.job.person}</dd>
											</div>
											<div>
												<dt>File</dt>
												<dd class="filename">{printer.job.file}</dd>
											</div>
											<div>
												<dt>Material</dt>
												<dd>{printer.job.material}</dd>
											</div>
											<div>
												<dt>Started</dt>
												<dd>{printer.job.started}</dd>
											</div>
											{#if printer.progress !== undefined}<div>
													<dt>Progress</dt>
													<dd class="detail-progress">
														<progress
															max="100"
															value={printer.progress}
															aria-label={`${printer.name} print progress`}
														></progress><span>{printer.progress}%</span>
													</dd>
												</div>{/if}
										</dl>
									{:else if printer.activity === 'unknown'}<p class="detail-empty">
											Current print details are unavailable.
										</p>
									{:else}<p class="detail-empty">No current print.</p>{/if}
									<p class="machine-reference">
										Machine ID: {printer.machineId ?? 'Not yet recorded'}
									</p>
								</section></td
							></tr
						>
					{/if}
				{:else}<tr
						><td colspan="6" class="table-empty"
							><p>
								{disconnected && ['printing', 'idle'].includes(filter)
									? 'Live activity is unavailable while updates are interrupted.'
									: 'No printers with this status.'}
							</p>
							<Button color="alternative" size="xs" on:click={() => (filter = 'all')}
								>Show all printers</Button
							></td
						></tr
					>{/each}
			</tbody>
		</table>
	</div>
	<footer>
		<p aria-live="polite">
			{visible.length} of {printers.length} printers · {disconnected
				? 'Updates unavailable'
				: `Updated ${updateTime(fetchedAt)}`}
		</p>
	</footer>
</main>

<style>
	.printer-dashboard {
		--ink: #252b35;
		--muted: #66707c;
		--line: #dee5e0;
		--paper: #fff;
		--subtle: #f5f7f8;
		--violet: #6943b9;
		--green: #246c48;
		--green-row: #f0f8f2;
		--green-badge: #dcefe1;
		--amber: #855d0d;
		--amber-row: #fff9eb;
		--amber-badge: #f7e9c4;
		--red: #a44343;
		--red-row: #fff4f2;
		--red-badge: #f7e0dc;
		color: var(--ink);
		max-width: 1600px;
		margin: 0 auto;
		padding: 0 8px 12px;
		font-family: Inter, ui-sans-serif, system-ui, sans-serif;
	}
	.page-heading {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 14px;
		padding: 2px 0 12px;
	}
	.title-line {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
	}
	h1 {
		font-size: 23px;
		letter-spacing: -0.035em;
		font-weight: 650;
		line-height: 1.25;
	}
	.sample-label {
		color: var(--muted);
		font-size: 10px;
		border: 1px solid var(--line);
		border-radius: 4px;
		padding: 2px 6px;
		white-space: nowrap;
	}
	.toolbar {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: center;
		padding: 0 0 12px;
		flex-wrap: wrap;
	}
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}
	:global(.filter-button) {
		color: var(--muted);
		padding: 6px 9px !important;
		border-radius: 5px !important;
		font-size: 11px !important;
		gap: 7px;
		border: 1px solid transparent;
	}
	:global(.filter-button:hover) {
		background: var(--subtle);
	}
	:global(.filter-button.selected) {
		background: var(--ink);
		color: var(--paper);
	}
	.filter-count {
		font-size: 10px;
		opacity: 0.8;
		font-variant-numeric: tabular-nums;
	}
	:global(.reset-sort) {
		color: var(--violet);
		font-size: 11px !important;
		padding: 4px !important;
	}
	.connection-banner {
		font-size: 12px;
		border: 1px solid #e8dbb7;
		background: #fbf5e8;
		color: #7e602a;
		padding: 9px 12px;
		border-radius: 5px;
		margin-bottom: 12px;
		line-height: 1.5;
	}
	.connection-banner strong {
		font-weight: 650;
		margin-right: 8px;
	}
	.fleet-table-scroll {
		overflow-x: auto;
		border-top: 1px solid var(--line);
	}
	.fleet-table-scroll:focus-within {
		outline: 2px solid var(--violet);
		outline-offset: 3px;
	}
	.fleet-table {
		width: 100%;
		min-width: 690px;
		table-layout: fixed;
		border-collapse: collapse;
		font-size: 12px;
		line-height: 1.45;
	}
	.printer-column {
		width: 218px;
	}
	.model-column {
		width: 72px;
	}
	.condition-column {
		width: 146px;
	}
	.activity-column {
		width: 90px;
	}
	.time-column {
		width: 120px;
	}
	.note-column {
		width: auto;
	}
	.fleet-table th,
	.fleet-table td {
		text-align: left;
		padding: 7px 10px;
		border-bottom: 1px solid var(--line);
		vertical-align: middle;
	}
	.fleet-table thead th {
		color: var(--muted);
		background: var(--subtle);
		font-size: 10px;
		font-weight: 600;
		padding: 0 7px;
	}
	.static-heading {
		display: inline-block;
		padding: 10px 3px;
	}
	.sort-button {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 10px 3px;
		text-align: left;
		white-space: nowrap;
		width: 100%;
	}
	.sort-button:hover,
	.sort-active {
		color: var(--ink);
	}
	.sort-arrow {
		opacity: 0.45;
	}
	.sort-active .sort-arrow {
		color: var(--violet);
		opacity: 1;
	}
	.sort-button:focus-visible,
	.printer-expand:focus-visible {
		outline: 2px solid var(--violet);
		outline-offset: -2px;
		border-radius: 3px;
	}
	.fleet-table tbody th {
		font-weight: 600;
		border-left: 3px solid transparent;
		padding-left: 8px;
	}
	.fleet-table .row-working {
		background: var(--green-row);
	}
	.fleet-table .row-working th {
		border-left-color: #3d8a5e;
	}
	.fleet-table .row-limited {
		background: var(--amber-row);
	}
	.fleet-table .row-limited th {
		border-left-color: #bf8d2b;
	}
	.fleet-table .row-out {
		background: var(--red-row);
	}
	.fleet-table .row-out th {
		border-left-color: #c06a62;
	}
	.fleet-table .row-stale {
		background: var(--subtle);
	}
	.fleet-table .row-stale th {
		border-left-color: #99a1ab;
	}
	.table-name {
		white-space: nowrap;
	}
	.table-model {
		color: var(--muted);
		font-size: 11px;
		white-space: nowrap;
	}
	.condition-badge {
		font-size: 11px;
		font-weight: 600;
		display: inline-flex;
		align-items: center;
		gap: 5px;
		line-height: 1.4;
		border-radius: 4px;
		padding: 3px 6px;
	}
	.condition-badge > span {
		width: 10px;
		text-align: center;
	}
	.working {
		color: var(--green);
		background: var(--green-badge);
	}
	.limited {
		color: var(--amber);
		background: var(--amber-badge);
	}
	.out {
		color: var(--red);
		background: var(--red-badge);
	}
	.unknown {
		color: var(--muted);
	}
	.table-printing,
	.table-time {
		color: var(--violet);
	}
	.table-time {
		font-variant-numeric: tabular-nums;
	}
	.table-muted {
		color: var(--muted);
	}
	.table-note {
		font-size: 12px;
		white-space: pre-line;
		overflow-wrap: anywhere;
	}
	.row-limited .table-note {
		color: var(--amber);
		font-weight: 550;
	}
	.row-out .table-note {
		color: var(--red);
	}
	.printer-expand {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		text-align: left;
		margin: -5px 0;
		padding: 5px 0;
		width: 100%;
	}
	.printer-expand:hover .table-name {
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.chevron {
		font-size: 20px;
		line-height: 15px;
		color: var(--muted);
		width: 9px;
		flex-shrink: 0;
	}
	.chevron.expanded {
		transform: rotate(90deg);
	}
	.fleet-table .details-row > td {
		padding: 0;
		background: var(--paper);
	}
	.print-details {
		padding: 14px 18px;
		border-left: 3px solid var(--violet);
	}
	.detail-heading {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: center;
		margin-bottom: 12px;
	}
	.detail-heading strong {
		font-size: 12px;
		font-weight: 600;
	}
	:global(.close-details) {
		padding: 3px 6px !important;
		color: var(--muted);
		font-size: 11px !important;
	}
	.detail-fields {
		display: flex;
		flex-wrap: wrap;
		gap: 16px 34px;
	}
	.detail-fields dt {
		font-size: 10px;
		color: var(--muted);
		margin-bottom: 3px;
	}
	.detail-fields dd {
		font-size: 12px;
	}
	.detail-fields .filename {
		font-family: ui-monospace, monospace;
		font-size: 11px;
		overflow-wrap: anywhere;
	}
	.detail-progress {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.detail-progress span {
		font-variant-numeric: tabular-nums;
	}
	progress {
		width: 80px;
		height: 4px;
		accent-color: var(--violet);
	}
	.machine-reference {
		font-size: 10px;
		color: var(--muted);
		margin-top: 12px;
	}
	.detail-empty {
		color: var(--muted);
		font-size: 12px;
	}
	.table-empty p {
		margin: 12px 0;
	}
	.fleet-table .table-empty {
		padding-bottom: 18px;
	}
	footer {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 9px;
		margin-top: 14px;
		padding-top: 10px;
		border-top: 1px solid var(--line);
		font-size: 10px;
		color: var(--muted);
	}
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	:global(.dark) .printer-dashboard {
		--ink: #ebedf2;
		--muted: #a6afbd;
		--line: #394353;
		--paper: #1f2937;
		--subtle: #263241;
		--violet: #c1a1f4;
		--green: #a2deba;
		--green-row: #23382e;
		--green-badge: #2e513d;
		--amber: #ebc983;
		--amber-row: #3c3426;
		--amber-badge: #54452a;
		--red: #f0b4ac;
		--red-row: #3c2c30;
		--red-badge: #59373a;
	}
	:global(.dark) .connection-banner {
		background: #3c3426;
		color: #ead4a5;
		border-color: #5a4930;
	}
	@media (max-width: 1100px) {
		.printer-column {
			width: 200px;
		}
		.model-column {
			width: 65px;
		}
		.condition-column {
			width: 130px;
		}
		.activity-column {
			width: 76px;
		}
		.time-column {
			width: 105px;
		}
	}
	@media (max-width: 760px) {
		.printer-column {
			width: 180px;
		}
		.model-column {
			width: 55px;
		}
		.condition-column {
			width: 125px;
		}
		.activity-column {
			width: 65px;
		}
		.time-column {
			width: 100px;
		}
		.fleet-table th,
		.fleet-table td {
			padding-left: 7px;
			padding-right: 7px;
		}
		.page-heading {
			flex-wrap: wrap;
			gap: 8px;
		}
		h1 {
			font-size: 21px;
		}
		.printer-dashboard {
			padding-left: 0;
			padding-right: 0;
		}
		.toolbar {
			gap: 8px;
		}
	}
	@media (max-width: 470px) {
		h1 {
			font-size: 20px;
		}
		.page-heading {
			padding-bottom: 10px;
		}
	}
</style>
