<script lang="ts">
	import type { ActivityPulse } from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	// Share of the week's visitors with no linked card above which the line turns amber.
	const unlinkedWarnPercent = 25;

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

	const activity = data.activity;

	// The four groups add up to a card's visitor total. "Card not linked" shares
	// its amber with the line below the cards, which is about the same people.
	function visitorGroups(window: ActivityPulse) {
		return [
			{ label: 'New members', value: window.new_visitors, swatch: 'swatch-new' },
			{ label: 'Returning members', value: window.returning_visitors, swatch: 'swatch-returning' },
			{ label: 'Student staff', value: window.staff_visitors, swatch: 'swatch-staff' },
			{ label: 'Card not linked', value: window.unknown_visitors, swatch: 'swatch-unlinked' }
		];
	}
	const pulse = activity.pulse ?? [];
	const unlinked = activity.still_unlinked;

	const heatAverages = new Map<string, number>();
	for (const cell of activity.heatmap) {
		const openDays = activity.heatmap_open_days?.[cell.weekday] ?? 0;
		if (openDays > 0) heatAverages.set(`${cell.weekday}-${cell.hour}`, cell.taps / openDays);
	}
	const heatMax = Math.max(1, ...heatAverages.values());
	const busiest = [...heatAverages.entries()].sort((a, b) => b[1] - a[1])[0];

	// The standard yellow-orange-red heatmap scale (ColorBrewer YlOrRd).
	const heatStops: [number, [number, number, number]][] = [
		[0, [255, 255, 204]],
		[0.25, [254, 217, 118]],
		[0.5, [253, 141, 60]],
		[0.75, [227, 26, 28]],
		[1, [128, 0, 38]]
	];

	function heatBackground(value: number): string {
		if (value <= 0) return 'transparent';
		const share = Math.min(1, value / heatMax);
		let upper = heatStops.findIndex(([stop]) => share <= stop);
		if (upper <= 0) upper = 1;
		const [fromStop, from] = heatStops[upper - 1];
		const [toStop, to] = heatStops[upper];
		const mix = (share - fromStop) / (toStop - fromStop);
		const channel = (index: number) => Math.round(from[index] + (to[index] - from[index]) * mix);
		return `rgb(${channel(0)}, ${channel(1)}, ${channel(2)})`;
	}

	function heatIsDark(value: number): boolean {
		return value / heatMax >= 0.65;
	}

	function heatAverage(weekday: number, hour: number): number {
		return heatAverages.get(`${weekday}-${hour}`) ?? 0;
	}

	function hourLabel(hour: number): string {
		if (hour === 12) return '12p';
		return hour < 12 ? `${hour}a` : `${hour - 12}p`;
	}

	function snapshotLabel(value: string): string {
		return new Intl.DateTimeFormat('en-US', {
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		}).format(new Date(value));
	}
</script>

<svelte:head><title>Activity · mkr.cx</title></svelte:head>

<main class="mx-auto flex w-full max-w-5xl flex-col gap-4 px-2 pb-6 md:px-6">
	<header>
		<h1 class="text-2xl font-bold text-gray-950 dark:text-white">Activity</h1>
		{#if activity.snapshot_at}
			<p class="mt-0.5 text-sm text-gray-600 dark:text-gray-300">
				As of {snapshotLabel(activity.snapshot_at)}
			</p>
		{/if}
	</header>

	<section aria-labelledby="pulse-heading">
		<h2 id="pulse-heading" class="sr-only">Visitors</h2>
		<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
			{#each pulse as window}
				{@const groups = visitorGroups(window)}
				<article
					class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
				>
					<h3 class="text-base font-bold text-gray-950 dark:text-white">{window.label}</h3>
					<strong class="mt-2 block text-4xl font-bold tabular-nums text-gray-950 dark:text-white"
						>{window.visitors.toLocaleString()}</strong
					>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
						visitors{#if (window.key === '7_days' || window.key === '30_days') && window.open_days > 0},
							about {Math.round(window.avg_daily_people).toLocaleString()} a day{/if}
					</p>
					<div
						class="mt-3 flex h-3 gap-px overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800"
						aria-hidden="true"
					>
						{#each groups as group}
							{#if group.value > 0}
								<span class={group.swatch} style="flex: {group.value} 1 0%;"></span>
							{/if}
						{/each}
					</div>
					<dl class="mt-2 flex flex-col gap-1 text-sm">
						{#each groups as group}
							<div class="flex items-center gap-2">
								<span class="h-3 w-3 shrink-0 rounded-sm {group.swatch}" aria-hidden="true"></span>
								<dt class="grow text-gray-600 dark:text-gray-300">{group.label}</dt>
								<dd class="font-semibold tabular-nums text-gray-950 dark:text-white">
									{group.value.toLocaleString()}
								</dd>
							</div>
						{/each}
					</dl>
				</article>
			{/each}
		</div>
	</section>

	{#if unlinked && unlinked.visitors > 0}
		<p
			class="rounded-2xl px-4 py-3 text-sm {unlinked.percent >= unlinkedWarnPercent
				? 'bg-amber-50 text-amber-900 dark:bg-amber-950 dark:text-amber-100'
				: 'bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-200'}"
		>
			<strong class="tabular-nums">{Math.round(unlinked.percent)}%</strong>
			({unlinked.cards.toLocaleString()}/{unlinked.visitors.toLocaleString()}) of visitors in the
			past 7 days have not linked their card.
		</p>
	{/if}

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="heat-heading"
	>
		<h2 id="heat-heading" class="text-lg font-bold text-gray-950 dark:text-white">
			When are we busy?
		</h2>
		<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
			Average card taps per hour, {activity.range.label}.
			{#if busiest}
				{@const [weekday, hour] = busiest[0].split('-').map(Number)}
				Busiest: {weekdays.find((day) => day.value === weekday)?.label}
				{hourLabel(hour)}.
			{/if}
		</p>
		<div class="mt-3 overflow-x-auto">
			<table class="w-full min-w-[34rem] border-separate border-spacing-0.5 text-center text-xs">
				<thead>
					<tr>
						<th class="w-10"></th>
						{#each hours as hour}
							<th class="font-medium text-gray-500 dark:text-gray-400">{hourLabel(hour)}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each weekdays as day}
						<tr>
							<th class="pr-1 text-right font-medium text-gray-500 dark:text-gray-400"
								>{day.label}</th
							>
							{#each hours as hour}
								{@const value = heatAverage(day.value, hour)}
								<td
									class="h-7 rounded tabular-nums {heatIsDark(value)
										? 'text-white'
										: 'text-gray-900'}"
									style="background-color: {heatBackground(value)};"
									title="{day.label} {hourLabel(hour)}: {value.toFixed(1)} card taps"
									>{value >= 0.5 ? Math.round(value) : ''}</td
								>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="years-heading"
	>
		<h2 id="years-heading" class="text-lg font-bold text-gray-950 dark:text-white">
			By academic year
		</h2>
		<div class="mt-2 grid gap-2 sm:grid-cols-3">
			{#each [...activity.academic_years].reverse() as year}
				<div
					class="rounded-xl px-4 py-3 {year.current
						? 'border-2 border-[#840028] bg-white dark:border-[#e07a9a] dark:bg-gray-900'
						: 'border-2 border-transparent bg-gray-50 dark:bg-gray-800'}"
				>
					<h3 class="text-base font-bold text-gray-950 dark:text-white">
						{year.label}{#if year.current}<span
								class="ml-1 text-sm font-medium text-gray-600 dark:text-gray-300">so far</span
							>{/if}
					</h3>
					<dl class="mt-2 flex flex-col gap-1 text-sm">
						<div class="flex items-baseline justify-between gap-2">
							<dt class="text-gray-600 dark:text-gray-300">Visitors</dt>
							<dd class="text-xl font-bold tabular-nums text-gray-950 dark:text-white">
								{year.visitors > 0
									? `${year.visitors_estimated ? 'about ' : ''}${year.visitors.toLocaleString()}`
									: '–'}
							</dd>
						</div>
						<div class="flex items-baseline justify-between gap-2">
							<dt class="text-gray-600 dark:text-gray-300">New registrations</dt>
							<dd class="text-xl font-bold tabular-nums text-gray-950 dark:text-white">
								{year.new_accounts.toLocaleString()}
							</dd>
						</div>
					</dl>
				</div>
			{/each}
		</div>
	</section>
</main>

<style>
	/* Calm on purpose: two blues for members, gray for staff, and amber only for
	   the group the amber line below is about. */
	.swatch-new {
		background-color: #38bdf8;
	}
	.swatch-returning {
		background-color: #1e40af;
	}
	.swatch-staff {
		background-color: #94a3b8;
	}
	.swatch-unlinked {
		background-color: #f59e0b;
	}
	:global(.dark) .swatch-returning {
		background-color: #3b82f6;
	}
</style>
