<script lang="ts">
	import type {
		ActivityPoint,
		ActivityRangeKey,
		ActivityResponse,
		ActivitySummary
	} from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	type TrendMode = 'daily' | 'weekly' | 'cumulative';
	type DisplayPoint = ActivityPoint & { value: number };
	type HeatPoint = { weekday: number; hour: number; members: number };
	type SummaryMetric = {
		label: string;
		description: string;
		field: keyof Pick<
			ActivitySummary,
			'visitors' | 'unlinked_card_holders' | 'new_accounts' | 'newly_linked_cards'
		>;
		minimum?: boolean;
	};

	const referenceDate = new Date(`${data.activity.range.end}T12:00:00`);
	const referenceYear = referenceDate.getFullYear();
	const academicStartYear = referenceDate.getMonth() >= 7 ? referenceYear : referenceYear - 1;
	const semester =
		referenceDate.getMonth() <= 4 ? 'Spring' : referenceDate.getMonth() <= 6 ? 'Summer' : 'Fall';
	const rangeOptions: { value: ActivityRangeKey; label: string }[] = [
		{ value: 'semester', label: `${semester} ${referenceYear}` },
		{
			value: 'academic_year',
			label: `${academicStartYear}–${String(academicStartYear + 1).slice(-2)} academic year`
		},
		{ value: '30_days', label: 'Past 30 days' }
	];
	const trendOptions: { value: TrendMode; label: string }[] = [
		{ value: 'daily', label: 'Daily' },
		{ value: 'weekly', label: 'Weekly' },
		{ value: 'cumulative', label: 'Cumulative' }
	];
	const summaryMetrics: SummaryMetric[] = [
		{
			label: 'Members (with linked cards)',
			description: 'Distinct members who tapped while linked',
			field: 'visitors'
		},
		{
			label: 'Unlinked card holders',
			description: 'Known minimum; later-linked cards only',
			field: 'unlinked_card_holders',
			minimum: true
		},
		{ label: 'New members', description: 'Accounts created', field: 'new_accounts' },
		{
			label: 'Newly linked cards',
			description: 'First recorded link per member',
			field: 'newly_linked_cards'
		}
	];
	const weekdays = [
		{ value: 1, label: 'Mon' },
		{ value: 2, label: 'Tue' },
		{ value: 3, label: 'Wed' },
		{ value: 4, label: 'Thu' },
		{ value: 5, label: 'Fri' },
		{ value: 6, label: 'Sat' },
		{ value: 0, label: 'Sun' }
	];
	const hours = Array.from({ length: 15 }, (_, index) => index + 8);

	const activityCache = new Map<ActivityRangeKey, ActivityResponse>([['semester', data.activity]]);
	let summaryActivity: ActivityResponse = data.activity;
	let trendActivity: ActivityResponse = data.activity;
	let heatActivity: ActivityResponse = data.activity;
	let selectedTrendRange: ActivityRangeKey = trendActivity.range.key;
	let selectedHeatRange: ActivityRangeKey = heatActivity.range.key;
	let trendMode: TrendMode = 'daily';
	let hoveredPoint = -1;
	let pinnedPoint = -1;
	let hoveredHeat: HeatPoint | null = null;
	let pinnedHeat: HeatPoint | null = null;
	let trendLoading = false;
	let heatLoading = false;
	let failure = '';

	$: points = trendPoints(trendActivity, trendMode);
	$: chartMax = Math.max(1, ...points.map((point) => point.value));
	$: highlightedPoint =
		points[hoveredPoint] || points[pinnedPoint] || points[points.length - 1] || null;
	$: heatMax = Math.max(1, ...heatActivity.heatmap.map((cell) => cell.members));
	$: highlightedHeat = hoveredHeat || pinnedHeat || busiestHeatPoint(heatActivity);
	$: comparisonMax = Math.max(
		1,
		...summaryActivity.academic_years.flatMap((year) => [
			year.new_accounts,
			year.newly_linked_cards
		])
	);

	function trendPoints(response: ActivityResponse, mode: TrendMode): DisplayPoint[] {
		const source = mode === 'weekly' ? response.weekly : response.daily;
		return source.map((point) => ({
			...point,
			value: mode === 'cumulative' ? point.cumulative_visitors : point.visitors
		}));
	}

	function fullDate(value: string): string {
		return new Intl.DateTimeFormat('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		}).format(new Date(`${value}T12:00:00`));
	}

	function dateRange(start: string, end: string): string {
		return `${fullDate(start)}–${fullDate(end)}`;
	}

	function hourLabel(hour: number): string {
		if (hour === 12) return '12p';
		return hour < 12 ? `${hour}a` : `${hour - 12}p`;
	}

	function snapshotLabel(value: string): string {
		return new Intl.DateTimeFormat('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			timeZone: summaryActivity.timezone
		}).format(new Date(value));
	}

	function metricValue(summary: ActivitySummary, metric: SummaryMetric): string {
		const value = summary[metric.field].toLocaleString();
		return metric.minimum ? `${value}+` : value;
	}

	function pointDescription(point: DisplayPoint): string {
		const prefix =
			trendMode === 'weekly' ? 'Week of ' : trendMode === 'cumulative' ? 'Through ' : '';
		return `${prefix}${fullDate(point.start)}`;
	}

	function chooseTrend(mode: TrendMode): void {
		trendMode = mode;
		hoveredPoint = -1;
		pinnedPoint = -1;
	}

	function togglePoint(index: number): void {
		pinnedPoint = pinnedPoint === index ? -1 : index;
	}

	function heatValue(response: ActivityResponse, weekday: number, hour: number): number {
		return (
			response.heatmap.find((cell) => cell.weekday === weekday && cell.hour === hour)?.members || 0
		);
	}

	function busiestHeatPoint(response: ActivityResponse): HeatPoint | null {
		if (response.heatmap.length === 0) return null;
		return response.heatmap.reduce((busiest, cell) =>
			cell.members > busiest.members ? cell : busiest
		);
	}

	function heatPoint(weekday: number, hour: number): HeatPoint {
		return { weekday, hour, members: heatValue(heatActivity, weekday, hour) };
	}

	function heatPointLabel(point: HeatPoint): string {
		const day = weekdays.find((weekday) => weekday.value === point.weekday)?.label || '';
		return `${day} ${hourLabel(point.hour)}–${hourLabel(point.hour + 1)}`;
	}

	function toggleHeat(point: HeatPoint): void {
		pinnedHeat =
			pinnedHeat?.weekday === point.weekday && pinnedHeat.hour === point.hour ? null : point;
	}

	function heatStyle(value: number): string {
		if (value === 0) return 'background-color: rgba(132, 0, 40, 0.05)';
		const opacity = 0.18 + (value / heatMax) * 0.82;
		return `background-color: rgba(132, 0, 40, ${opacity.toFixed(2)})`;
	}

	async function loadActivity(range: ActivityRangeKey): Promise<ActivityResponse> {
		const cached = activityCache.get(range);
		if (cached) return cached;
		const response = await data.api.getActivity(range);
		activityCache.set(range, response);
		return response;
	}

	async function changeTrendRange(): Promise<void> {
		const requestedRange = selectedTrendRange;
		trendLoading = true;
		failure = '';
		hoveredPoint = -1;
		pinnedPoint = -1;
		trendMode = requestedRange === 'academic_year' ? 'weekly' : 'daily';
		try {
			const response = await loadActivity(requestedRange);
			if (selectedTrendRange === requestedRange) trendActivity = response;
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to load activity.';
		} finally {
			trendLoading = false;
		}
	}

	async function changeHeatRange(): Promise<void> {
		const requestedRange = selectedHeatRange;
		heatLoading = true;
		failure = '';
		hoveredHeat = null;
		pinnedHeat = null;
		try {
			const response = await loadActivity(requestedRange);
			if (selectedHeatRange === requestedRange) heatActivity = response;
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to load activity.';
		} finally {
			heatLoading = false;
		}
	}
</script>

<svelte:head><title>Activity · mkr.cx</title></svelte:head>

<main class="mx-auto flex w-full max-w-7xl flex-col gap-4 px-2 pb-8 md:px-6">
	<header>
		<div>
			<p class="text-sm font-semibold uppercase tracking-wide text-[#840028] dark:text-[#e07a9a]">
				Makerspace pulse
			</p>
			<h1 class="mt-1 text-3xl font-bold text-gray-950 dark:text-white">Activity</h1>
			<p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
				A current snapshot, two focused explorers, and a fixed academic-year comparison.
			</p>
		</div>
	</header>

	{#if failure}
		<div
			role="alert"
			class="rounded-xl border border-red-300 bg-red-50 p-3 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
		>
			{failure}
		</div>
	{/if}

	{#if summaryActivity.snapshot_at}
		<p
			class="rounded-xl bg-gray-100 px-4 py-2 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300"
		>
			<strong>Real production data</strong> · Snapshot through {snapshotLabel(
				summaryActivity.snapshot_at
			)}.
		</p>
	{/if}

	<section aria-labelledby="snapshot-heading">
		<div class="mb-2 flex flex-wrap items-end justify-between gap-2 px-1">
			<div>
				<h2 id="snapshot-heading" class="text-lg font-bold text-gray-950 dark:text-white">
					At a glance
				</h2>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					Today, this week, and {summaryActivity.range.label}. These figures do not change with the
					charts below.
				</p>
			</div>
			<span class="text-xs text-gray-500 dark:text-gray-400">
				{dateRange(summaryActivity.range.start, summaryActivity.range.end)}
			</span>
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			{#each summaryMetrics as metric}
				<article
					class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
				>
					<div class="flex items-baseline justify-between gap-3">
						<h2 class="font-semibold text-gray-950 dark:text-white">{metric.label}</h2>
						{#if metric.minimum}<span class="text-xs font-medium text-[#840028] dark:text-[#e07a9a]"
								>known minimum</span
							>{/if}
					</div>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{metric.description}</p>
					<div
						class="mt-3 grid grid-cols-3 divide-x divide-gray-200 border-t border-gray-100 pt-3 text-center dark:divide-gray-700 dark:border-gray-800"
					>
						{#each [{ label: 'Today', value: summaryActivity.today }, { label: 'This week', value: summaryActivity.week }, { label: summaryActivity.range.label, value: summaryActivity.selected }] as period}
							<div class="px-2 first:pl-0 last:pr-0">
								<strong class="block text-2xl font-bold tabular-nums text-gray-950 dark:text-white"
									>{metricValue(period.value, metric)}</strong
								>
								<span
									class="mt-0.5 block text-[11px] leading-tight text-gray-500 dark:text-gray-400"
									>{period.label}</span
								>
							</div>
						{/each}
					</div>
				</article>
			{/each}
		</div>
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="trend-heading"
	>
		<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
			<div>
				<h2 id="trend-heading" class="text-lg font-bold text-gray-950 dark:text-white">
					Members with linked cards
				</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
					Explore one period at a time. Each person is counted once per bar.
				</p>
			</div>
			<div class="flex flex-wrap items-end gap-3">
				<label class="flex flex-col gap-1 text-xs font-medium text-gray-600 dark:text-gray-300">
					Chart period
					<select
						bind:value={selectedTrendRange}
						on:change={changeTrendRange}
						disabled={trendLoading}
						class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-950 focus:border-[#840028] focus:ring-[#840028] disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-white"
					>
						{#each rangeOptions as option}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</label>
				<div>
					<span class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300">Bars</span>
					<div
						class="inline-flex w-fit rounded-lg bg-gray-100 p-1 dark:bg-gray-800"
						aria-label="Trend grouping"
					>
						{#each trendOptions as option}
							<button
								type="button"
								on:click={() => chooseTrend(option.value)}
								class="rounded-md px-3 py-1 text-sm font-medium transition {trendMode ===
								option.value
									? 'bg-[#840028] text-white shadow-sm'
									: 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
								aria-pressed={trendMode === option.value}>{option.label}</button
							>
						{/each}
					</div>
				</div>
			</div>
		</div>

		{#if points.length > 0 && highlightedPoint}
			<div
				class="mt-3 flex items-baseline justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-gray-800"
			>
				<span class="text-sm text-gray-600 dark:text-gray-300"
					>{pointDescription(highlightedPoint)}</span
				>
				<strong class="text-lg tabular-nums text-gray-950 dark:text-white"
					>{highlightedPoint.value.toLocaleString()}
					{highlightedPoint.value === 1 ? 'member' : 'members'}</strong
				>
			</div>
			<div class="mt-3">
				<div
					class="grid h-36 items-end gap-px border-b border-gray-200 dark:border-gray-700"
					style={`grid-template-columns: repeat(${points.length}, minmax(0, 1fr));`}
				>
					{#each points as point, index}
						<button
							type="button"
							class="group flex h-full min-w-0 items-end focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#840028]"
							on:mouseenter={() => (hoveredPoint = index)}
							on:mouseleave={() => (hoveredPoint = -1)}
							on:focus={() => (hoveredPoint = index)}
							on:blur={() => (hoveredPoint = -1)}
							on:click={() => togglePoint(index)}
							aria-label={`${pointDescription(point)}: ${point.value} members`}
							aria-pressed={pinnedPoint === index}
						>
							<span
								class="block w-full rounded-t-[2px] bg-[#840028] transition group-hover:bg-[#a3274e] group-focus-visible:bg-[#a3274e]"
								style={`height: ${point.value === 0 ? 2 : Math.max(4, (point.value / chartMax) * 100)}%`}
							></span>
						</button>
					{/each}
				</div>
				<div class="mt-1 flex justify-between text-[10px] text-gray-500 dark:text-gray-400">
					<span>{fullDate(points[0].start)}</span>
					<span>{fullDate(points[points.length - 1].start)}</span>
				</div>
			</div>
		{:else}
			<p
				class="mt-4 rounded-xl bg-gray-50 p-4 text-center text-gray-500 dark:bg-gray-800 dark:text-gray-400"
			>
				No linked-member activity recorded in this range yet.
			</p>
		{/if}
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="heat-heading"
	>
		<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
			<div>
				<h2 id="heat-heading" class="text-lg font-bold text-gray-950 dark:text-white">
					When members arrive
				</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
					Choose a period, then hover or select a square. This shows distinct identifiable members,
					not occupancy.
				</p>
			</div>
			<label class="flex w-fit flex-col gap-1 text-xs font-medium text-gray-600 dark:text-gray-300">
				Heatmap period
				<select
					bind:value={selectedHeatRange}
					on:change={changeHeatRange}
					disabled={heatLoading}
					class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-950 focus:border-[#840028] focus:ring-[#840028] disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-white"
				>
					{#each rangeOptions as option}
						<option value={option.value}>{option.label}</option>
					{/each}
				</select>
			</label>
		</div>

		{#if highlightedHeat}
			<div
				class="mt-3 flex items-baseline justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-gray-800"
			>
				<span class="text-sm text-gray-600 dark:text-gray-300"
					>{heatPointLabel(highlightedHeat)}</span
				>
				<strong class="text-lg tabular-nums text-gray-950 dark:text-white">
					{highlightedHeat.members.toLocaleString()}
					{highlightedHeat.members === 1 ? 'member' : 'members'}
				</strong>
			</div>
		{/if}

		<div class="mt-3 overflow-x-auto pb-1">
			<div
				class="grid min-w-[42rem] gap-1"
				style="grid-template-columns: 2.25rem repeat(15, minmax(1.1rem, 1fr));"
			>
				<div></div>
				{#each hours as hour}
					<div class="pb-0.5 text-center text-[9px] font-medium text-gray-500 dark:text-gray-400">
						{hour % 2 === 0 ? hourLabel(hour) : ''}
					</div>
				{/each}
				{#each weekdays as weekday}
					<div class="flex items-center text-[11px] font-semibold text-gray-600 dark:text-gray-300">
						{weekday.label}
					</div>
					{#each hours as hour}
						{@const point = heatPoint(weekday.value, hour)}
						<button
							type="button"
							class="flex h-7 min-w-0 items-center justify-center rounded text-[9px] font-semibold outline-none transition focus-visible:ring-2 focus-visible:ring-[#840028] focus-visible:ring-offset-1 {point.members /
								heatMax >
							0.42
								? 'text-white'
								: 'text-gray-700 dark:text-gray-200'} {pinnedHeat?.weekday === weekday.value &&
							pinnedHeat.hour === hour
								? 'ring-2 ring-[#840028] ring-offset-1'
								: ''}"
							style={heatStyle(point.members)}
							on:mouseenter={() => (hoveredHeat = point)}
							on:mouseleave={() => (hoveredHeat = null)}
							on:focus={() => (hoveredHeat = point)}
							on:blur={() => (hoveredHeat = null)}
							on:click={() => toggleHeat(point)}
							aria-label={`${heatPointLabel(point)}: ${point.members} identifiable members`}
							aria-pressed={pinnedHeat?.weekday === weekday.value && pinnedHeat.hour === hour}
						>
							{point.members || ''}
						</button>
					{/each}
				{/each}
			</div>
		</div>
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="comparison-heading"
	>
		<div class="flex flex-wrap items-start justify-between gap-2">
			<div>
				<h2 id="comparison-heading" class="text-lg font-bold text-gray-950 dark:text-white">
					Academic-year comparison
				</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
					Membership activity from existing records.
				</p>
			</div>
			<span
				class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300"
			>
				Fixed comparison
			</span>
		</div>
		<div class="mt-4 grid gap-4 md:grid-cols-3">
			{#each summaryActivity.academic_years as year}
				<div class="rounded-xl border border-gray-100 p-3 dark:border-gray-800">
					<div class="flex flex-wrap items-baseline justify-between gap-2">
						<strong class="text-sm text-gray-900 dark:text-white">
							{year.label}{year.current ? ' to date' : ''}
						</strong>
						<span class="text-[10px] text-gray-500 dark:text-gray-400">
							{dateRange(year.start, year.end)}
						</span>
					</div>
					<div class="mt-3 grid grid-cols-[6.5rem_1fr_3rem] items-center gap-2 text-xs">
						<span class="text-gray-600 dark:text-gray-300">New members</span>
						<div class="h-2 rounded-full bg-gray-100 dark:bg-gray-800">
							<div
								class="h-2 rounded-full bg-[#840028]"
								style={`width: ${(year.new_accounts / comparisonMax) * 100}%`}
							></div>
						</div>
						<strong class="text-right tabular-nums text-gray-900 dark:text-white">
							{year.new_accounts.toLocaleString()}
						</strong>
						<span class="text-gray-600 dark:text-gray-300">New card links</span>
						<div class="h-2 rounded-full bg-gray-100 dark:bg-gray-800">
							<div
								class="h-2 rounded-full bg-gray-700 dark:bg-gray-300"
								style={`width: ${(year.newly_linked_cards / comparisonMax) * 100}%`}
							></div>
						</div>
						<strong class="text-right tabular-nums text-gray-900 dark:text-white">
							{year.newly_linked_cards.toLocaleString()}
						</strong>
					</div>
				</div>
			{/each}
		</div>
		{#if summaryActivity.coverage.first_card_link}
			<p
				class="mt-4 border-t border-gray-100 pt-3 text-[11px] text-gray-500 dark:border-gray-800 dark:text-gray-400"
			>
				Recorded card-link history begins {fullDate(summaryActivity.coverage.first_card_link)}.
			</p>
		{/if}
	</section>

	<p class="px-1 text-xs text-gray-500 dark:text-gray-400">
		Linked-member trends use card-tap history beginning {summaryActivity.coverage.first_checkin
			? fullDate(summaryActivity.coverage.first_checkin)
			: 'with the current data set'}. Unlinked-card figures are minimums because older unresolved
		cards cannot be deduplicated.
	</p>
</main>
