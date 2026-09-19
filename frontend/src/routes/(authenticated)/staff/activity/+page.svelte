<script lang="ts">
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
	const pulse = activity.pulse ?? [];
	const unlinked = activity.still_unlinked;

	const heatAverages = new Map<string, number>();
	for (const cell of activity.heatmap) {
		const openDays = activity.heatmap_open_days?.[cell.weekday] ?? 0;
		if (openDays > 0) heatAverages.set(`${cell.weekday}-${cell.hour}`, cell.taps / openDays);
	}
	const heatMax = Math.max(1, ...heatAverages.values());
	const busiest = [...heatAverages.entries()].sort((a, b) => b[1] - a[1])[0];

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
		<div class="grid gap-2 sm:grid-cols-3">
			{#each pulse as window}
				<article
					class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
				>
					<h3 class="text-sm font-semibold text-gray-600 dark:text-gray-300">{window.label}</h3>
					<strong class="mt-1 block text-5xl font-bold tabular-nums text-gray-950 dark:text-white"
						>{window.visitors.toLocaleString()}</strong
					>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">visitors</p>
					<dl
						class="mt-3 grid grid-cols-3 gap-2 border-t border-gray-100 pt-2 text-sm dark:border-gray-800"
					>
						{#each [{ label: 'New', value: window.new_visitors }, { label: 'Returning', value: window.returning_visitors }, { label: 'Unknown', value: window.unknown_visitors }] as part}
							<div>
								<dt class="text-xs text-gray-500 dark:text-gray-400">{part.label}</dt>
								<dd class="font-semibold tabular-nums text-gray-950 dark:text-white">
									{part.value.toLocaleString()}
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
			past 7 days still have no linked card.
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
									class="h-7 rounded tabular-nums {value / heatMax > 0.4
										? 'text-white'
										: 'text-gray-700 dark:text-gray-200'}"
									style="background-color: rgba(132, 0, 40, {value > 0
										? 0.08 + 0.92 * (value / heatMax)
										: 0});"
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
			New registrations by year
		</h2>
		<dl class="mt-2 grid grid-cols-3 gap-2 text-center">
			{#each activity.academic_years as year}
				<div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-gray-800">
					<dt class="text-xs text-gray-500 dark:text-gray-400">
						{year.label}{year.current ? ' so far' : ''}
					</dt>
					<dd class="text-2xl font-bold tabular-nums text-gray-950 dark:text-white">
						{year.new_accounts.toLocaleString()}
					</dd>
				</div>
			{/each}
		</dl>
	</section>
</main>
