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

	let activity: ActivityResponse = data.activity;
	let selectedRange: ActivityRangeKey = activity.range.key;
	let trendMode: TrendMode = 'daily';
	let hoveredPoint = -1;
	let pinnedPoint = -1;
	let loading = false;
	let failure = '';

	$: points = trendPoints(activity, trendMode);
	$: chartMax = Math.max(1, ...points.map((point) => point.value));
	$: highlightedPoint =
		points[hoveredPoint] || points[pinnedPoint] || points[points.length - 1] || null;
	$: heatMax = Math.max(1, ...activity.heatmap.map((cell) => cell.members));
	$: comparisonMax = Math.max(
		1,
		...activity.academic_years.flatMap((year) => [year.new_accounts, year.newly_linked_cards])
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
			timeZone: activity.timezone
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

	function heatValue(weekday: number, hour: number): number {
		return (
			activity.heatmap.find((cell) => cell.weekday === weekday && cell.hour === hour)?.members || 0
		);
	}

	function heatStyle(value: number): string {
		if (value === 0) return 'background-color: rgba(132, 0, 40, 0.05)';
		const opacity = 0.18 + (value / heatMax) * 0.82;
		return `background-color: rgba(132, 0, 40, ${opacity.toFixed(2)})`;
	}

	async function changeRange(): Promise<void> {
		loading = true;
		failure = '';
		hoveredPoint = -1;
		pinnedPoint = -1;
		trendMode = selectedRange === 'academic_year' ? 'weekly' : 'daily';
		try {
			activity = await data.api.getActivity(selectedRange);
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to load activity.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Activity · mkr.cx</title></svelte:head>

<main class="mx-auto flex w-full max-w-7xl flex-col gap-4 px-2 pb-8 md:px-6">
	<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p class="text-sm font-semibold uppercase tracking-wide text-[#840028] dark:text-[#e07a9a]">
				Makerspace pulse
			</p>
			<h1 class="mt-1 text-3xl font-bold text-gray-950 dark:text-white">Activity</h1>
			<p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
				{dateRange(activity.range.start, activity.range.end)}
			</p>
		</div>
		<label
			class="flex w-full flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200 sm:w-64"
		>
			Time range
			<select
				bind:value={selectedRange}
				on:change={changeRange}
				disabled={loading}
				class="rounded-xl border border-gray-300 bg-white px-3 py-2.5 text-gray-950 shadow-sm focus:border-[#840028] focus:ring-[#840028] disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-white"
			>
				{#each rangeOptions as option}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>
		</label>
	</header>

	{#if failure}
		<div
			role="alert"
			class="rounded-xl border border-red-300 bg-red-50 p-3 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
		>
			{failure}
		</div>
	{/if}

	{#if activity.snapshot_at}
		<p
			class="rounded-xl bg-gray-100 px-4 py-2 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300"
		>
			<strong>Real production data</strong> · Snapshot through {snapshotLabel(
				activity.snapshot_at
			)}.
		</p>
	{/if}

	<section aria-label="Activity summary" class="grid gap-3 md:grid-cols-2">
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
					{#each [{ label: 'Today', value: activity.today }, { label: 'This week', value: activity.week }, { label: activity.range.label, value: activity.selected }] as period}
						<div class="px-2 first:pl-0 last:pr-0">
							<strong class="block text-2xl font-bold tabular-nums text-gray-950 dark:text-white"
								>{metricValue(period.value, metric)}</strong
							>
							<span class="mt-0.5 block text-[11px] leading-tight text-gray-500 dark:text-gray-400"
								>{period.label}</span
							>
						</div>
					{/each}
				</div>
			</article>
		{/each}
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
					Each person is counted once per period.
				</p>
			</div>
			<div
				class="inline-flex w-fit rounded-lg bg-gray-100 p-1 dark:bg-gray-800"
				aria-label="Trend grouping"
			>
				{#each trendOptions as option}
					<button
						type="button"
						on:click={() => chooseTrend(option.value)}
						class="rounded-md px-3 py-1.5 text-sm font-medium transition {trendMode === option.value
							? 'bg-[#840028] text-white shadow-sm'
							: 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
						aria-pressed={trendMode === option.value}>{option.label}</button
					>
				{/each}
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

	<div class="grid gap-4 xl:grid-cols-[minmax(0,1.55fr)_minmax(20rem,0.75fr)]">
		<section
			class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
			aria-labelledby="heat-heading"
		>
			<h2 id="heat-heading" class="text-lg font-bold text-gray-950 dark:text-white">
				When members arrive
			</h2>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
				Distinct identifiable members by weekday and hour; not occupancy.
			</p>
			<div
				class="mt-4 grid gap-1"
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
						{@const value = heatValue(weekday.value, hour)}
						<div
							class="flex h-7 min-w-0 items-center justify-center rounded text-[9px] font-semibold {value /
								heatMax >
							0.42
								? 'text-white'
								: 'text-gray-700 dark:text-gray-200'}"
							style={heatStyle(value)}
							title={`${weekday.label} ${hourLabel(hour)}: ${value} identifiable members`}
							aria-label={`${weekday.label} ${hourLabel(hour)}, ${value} identifiable members`}
						>
							{value || ''}
						</div>
					{/each}
				{/each}
			</div>
		</section>

		<section
			class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
			aria-labelledby="comparison-heading"
		>
			<h2 id="comparison-heading" class="text-lg font-bold text-gray-950 dark:text-white">
				Academic-year comparison
			</h2>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
				Membership activity from existing records.
			</p>
			<div class="mt-4 space-y-4">
				{#each activity.academic_years as year}
					<div>
						<div class="flex items-baseline justify-between gap-3">
							<strong class="text-sm text-gray-900 dark:text-white"
								>{year.label}{year.current ? ' to date' : ''}</strong
							>
							<span class="text-[10px] text-gray-500 dark:text-gray-400"
								>{dateRange(year.start, year.end)}</span
							>
						</div>
						<div class="mt-1.5 grid grid-cols-[6.5rem_1fr_3rem] items-center gap-2 text-xs">
							<span class="text-gray-600 dark:text-gray-300">New members</span>
							<div class="h-2 rounded-full bg-gray-100 dark:bg-gray-800">
								<div
									class="h-2 rounded-full bg-[#840028]"
									style={`width: ${(year.new_accounts / comparisonMax) * 100}%`}
								></div>
							</div>
							<strong class="text-right tabular-nums text-gray-900 dark:text-white"
								>{year.new_accounts.toLocaleString()}</strong
							>
							<span class="text-gray-600 dark:text-gray-300">New card links</span>
							<div class="h-2 rounded-full bg-gray-100 dark:bg-gray-800">
								<div
									class="h-2 rounded-full bg-gray-700 dark:bg-gray-300"
									style={`width: ${(year.newly_linked_cards / comparisonMax) * 100}%`}
								></div>
							</div>
							<strong class="text-right tabular-nums text-gray-900 dark:text-white"
								>{year.newly_linked_cards.toLocaleString()}</strong
							>
						</div>
					</div>
				{/each}
			</div>
			{#if activity.coverage.first_card_link}
				<p
					class="mt-4 border-t border-gray-100 pt-3 text-[11px] text-gray-500 dark:border-gray-800 dark:text-gray-400"
				>
					Recorded card-link history begins {fullDate(activity.coverage.first_card_link)}.
				</p>
			{/if}
		</section>
	</div>

	<p class="px-1 text-xs text-gray-500 dark:text-gray-400">
		Linked-member trends use card-tap history beginning {activity.coverage.first_checkin
			? fullDate(activity.coverage.first_checkin)
			: 'with the current data set'}. Unlinked-card figures are minimums because older unresolved
		cards cannot be deduplicated.
	</p>
</main>
