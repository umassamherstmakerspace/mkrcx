<script lang="ts">
	import type { ActivityPulse } from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	// Share of people without a linked card above which the page asks staff to act.
	const notLinkedWarnPercent = 25;
	// Days without any tap before the page suggests checking the reader.
	const quietWarnDays = 4;

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
	const week = pulse.find((window) => window.key === '7_days');

	const heatAverages = new Map<string, number>();
	for (const cell of activity.heatmap) {
		const openDays = activity.heatmap_open_days?.[cell.weekday] ?? 0;
		if (openDays > 0) heatAverages.set(`${cell.weekday}-${cell.hour}`, cell.taps / openDays);
	}
	const heatMax = Math.max(1, ...heatAverages.values());
	const busiest = [...heatAverages.entries()].sort((a, b) => b[1] - a[1])[0];

	const lastTapDay = [...activity.daily].reverse().find((day) => day.checkins > 0)?.start;
	const quietDays = lastTapDay
		? Math.round(
				(new Date(`${activity.range.end}T12:00:00`).getTime() -
					new Date(`${lastTapDay}T12:00:00`).getTime()) /
					86_400_000
			)
		: null;

	function headline(window: ActivityPulse): string {
		if (window.key === 'today') return window.people.toLocaleString();
		return window.open_days > 0 ? Math.round(window.avg_daily_people).toLocaleString() : '–';
	}

	function headlineCaption(window: ActivityPulse): string {
		if (window.key === 'today') return 'people so far today';
		const days = `${window.open_days} open day${window.open_days === 1 ? '' : 's'}`;
		return `people per open day (${days})`;
	}

	function heatAverage(weekday: number, hour: number): number {
		return heatAverages.get(`${weekday}-${hour}`) ?? 0;
	}

	function hourLabel(hour: number): string {
		if (hour === 12) return '12p';
		return hour < 12 ? `${hour}a` : `${hour - 12}p`;
	}

	function fullDate(value: string): string {
		return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(
			new Date(`${value}T12:00:00`)
		);
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
		<p class="mt-0.5 text-sm text-gray-600 dark:text-gray-300">
			One person counts once per day, whether or not their card is linked.
			{#if activity.snapshot_at}Numbers as of {snapshotLabel(activity.snapshot_at)}.{/if}
		</p>
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
						>{headline(window)}</strong
					>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{headlineCaption(window)}</p>
					<dl
						class="mt-3 grid grid-cols-2 gap-2 border-t border-gray-100 pt-2 text-sm dark:border-gray-800"
					>
						<div>
							<dt class="text-xs text-gray-500 dark:text-gray-400">New members</dt>
							<dd class="font-semibold tabular-nums text-gray-950 dark:text-white">
								{window.new_accounts.toLocaleString()}
							</dd>
						</div>
						<div>
							<dt class="text-xs text-gray-500 dark:text-gray-400">Taps</dt>
							<dd class="font-semibold tabular-nums text-gray-950 dark:text-white">
								{window.checkins.toLocaleString()}
							</dd>
						</div>
					</dl>
				</article>
			{/each}
		</div>
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="signals-heading"
	>
		<h2 id="signals-heading" class="text-lg font-bold text-gray-950 dark:text-white">
			Anything to fix?
		</h2>
		<ul class="mt-2 flex flex-col gap-2 text-sm">
			{#if week}
				{@const warn = week.not_linked_percent >= notLinkedWarnPercent}
				<li
					class="rounded-lg px-3 py-2 {warn
						? 'bg-amber-50 text-amber-900 dark:bg-amber-950 dark:text-amber-100'
						: 'bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-200'}"
				>
					<strong class="tabular-nums">{Math.round(week.not_linked_percent)}%</strong> of people in
					the past 7 days tapped without a linked card ({week.not_linked_people.toLocaleString()}
					of {week.people.toLocaleString()}).
					{#if warn}Ask the front desk to help people link their UCard.{/if}
					<span class="text-gray-500 dark:text-gray-400">
						{week.newly_linked_cards.toLocaleString()} card{week.newly_linked_cards === 1
							? ''
							: 's'} newly linked in the same period.</span
					>
				</li>
			{/if}
			{#if quietDays !== null && quietDays >= quietWarnDays && lastTapDay}
				<li class="rounded-lg bg-amber-50 px-3 py-2 text-amber-900 dark:bg-amber-950 dark:text-amber-100">
					No taps since {fullDate(lastTapDay)}. If the space was open, check the card reader.
				</li>
			{:else if lastTapDay}
				<li class="rounded-lg bg-gray-50 px-3 py-2 text-gray-700 dark:bg-gray-800 dark:text-gray-200">
					Card reader is reporting. Last tap: {fullDate(lastTapDay)}.
				</li>
			{/if}
		</ul>
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
		aria-labelledby="heat-heading"
	>
		<h2 id="heat-heading" class="text-lg font-bold text-gray-950 dark:text-white">
			When are we busy?
		</h2>
		<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
			Average taps per hour, {activity.range.label}.
			{#if busiest}
				{@const [weekday, hour] = busiest[0].split('-').map(Number)}
				Busiest: {weekdays.find((day) => day.value === weekday)?.label}
				{hourLabel(hour)}, about {Math.round(busiest[1])} taps.
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
									class="h-7 rounded tabular-nums {value / heatMax > 0.55
										? 'text-white'
										: 'text-gray-700 dark:text-gray-200'}"
									style="background-color: rgba(132, 0, 40, {value > 0
										? 0.08 + 0.92 * (value / heatMax)
										: 0});"
									title="{day.label} {hourLabel(hour)}: about {value.toFixed(1)} taps"
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
			New members by academic year
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

	<p class="px-1 text-xs text-gray-500 dark:text-gray-400">
		Unknown cards are counted per day and only the count is kept. Days before that counting began
		show members only, so older averages are "at least" figures. For exact date ranges, use the
		check-in export.
	</p>
</main>
