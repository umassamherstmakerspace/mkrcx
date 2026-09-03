<script lang="ts">
	import { page } from '$app/stores';
	import type { ActivityPoint, ActivityRangeKey, ActivityResponse } from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	type TrendMode = 'daily' | 'weekly' | 'cumulative';
	type DisplayPoint = ActivityPoint & { value: number };

	const rangeOptions: { value: ActivityRangeKey; label: string }[] = [
		{ value: 'semester', label: 'This semester' },
		{ value: 'academic_year', label: 'This academic year' },
		{ value: '30_days', label: 'Past 30 days' }
	];
	const trendOptions: { value: TrendMode; label: string }[] = [
		{ value: 'daily', label: 'Daily' },
		{ value: 'weekly', label: 'Weekly' },
		{ value: 'cumulative', label: 'Cumulative' }
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

	let liveActivity: ActivityResponse = data.activity;
	let activity: ActivityResponse;
	let selectedRange: ActivityRangeKey = liveActivity.range.key;
	let trendMode: TrendMode = 'daily';
	let loading = false;
	let failure = '';

	$: previewing = $page.url.searchParams.get('preview') === '1';
	$: activity = previewing ? sampleActivity(liveActivity) : liveActivity;
	$: points = trendPoints(activity, trendMode);
	$: chartMax = Math.max(1, ...points.map((point) => point.value));
	$: heatMax = Math.max(1, ...activity.heatmap.map((cell) => cell.checkins));

	function trendPoints(response: ActivityResponse, mode: TrendMode): DisplayPoint[] {
		const source = mode === 'weekly' ? response.weekly : response.daily;
		return source.map((point) => ({
			...point,
			value: mode === 'cumulative' ? point.cumulative_visitors : point.visitors
		}));
	}

	function shortDate(value: string): string {
		return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(
			new Date(`${value}T12:00:00`)
		);
	}

	function hourLabel(hour: number): string {
		if (hour === 12) return '12p';
		return hour < 12 ? `${hour}a` : `${hour - 12}p`;
	}

	function heatValue(weekday: number, hour: number): number {
		return (
			activity.heatmap.find((cell) => cell.weekday === weekday && cell.hour === hour)?.checkins || 0
		);
	}

	function heatStyle(value: number): string {
		if (value === 0) return 'background-color: rgba(16, 185, 129, 0.06)';
		const opacity = 0.18 + (value / heatMax) * 0.72;
		return `background-color: rgba(5, 150, 105, ${opacity.toFixed(2)})`;
	}

	function sampleActivity(source: ActivityResponse): ActivityResponse {
		let cumulativeVisitors = 0;
		const daily = source.daily.map((point, index) => {
			const weekday = new Date(`${point.start}T12:00:00`).getDay();
			const openDay = weekday >= 1 && weekday <= 5;
			const wave = Math.round(5 * Math.sin(index / 4) + 3 * Math.cos(index / 9));
			const visitors = Math.max(0, openDay ? 19 + wave + (weekday === 3 ? 7 : 0) : 2 + wave);
			const checkins = Math.max(visitors, Math.round(visitors * (1.18 + (index % 4) * 0.06)));
			const newAccounts = openDay && index % 5 === 1 ? 2 : index % 6 === 0 ? 1 : 0;
			cumulativeVisitors += visitors;
			return {
				...point,
				visitors,
				checkins,
				new_accounts: newAccounts,
				cumulative_visitors: cumulativeVisitors
			};
		});

		let weeklyCumulative = 0;
		const weekly = source.weekly.map((point, index) => {
			const nextStart = source.weekly[index + 1]?.start;
			const days = daily.filter(
				(day) => day.start >= point.start && (!nextStart || day.start < nextStart)
			);
			const dailyVisitors = days.reduce((sum, day) => sum + day.visitors, 0);
			const visitors = Math.round(dailyVisitors * 0.78);
			weeklyCumulative += visitors;
			return {
				...point,
				visitors,
				checkins: days.reduce((sum, day) => sum + day.checkins, 0),
				new_accounts: days.reduce((sum, day) => sum + day.new_accounts, 0),
				cumulative_visitors: weeklyCumulative
			};
		});

		const lastDay = daily.at(-1) || {
			visitors: 23,
			checkins: 29,
			new_accounts: 1,
			start: source.range.end,
			cumulative_visitors: 23
		};
		const currentWeek = weekly.at(-1) || lastDay;
		const totalCheckins = daily.reduce((sum, day) => sum + day.checkins, 0);
		const totalAccounts = daily.reduce((sum, day) => sum + day.new_accounts, 0);

		const heatmap = weekdays.flatMap((weekday) =>
			hours.map((hour) => {
				const weekdayFactor = weekday.value >= 1 && weekday.value <= 5 ? 1 : 0.12;
				const midday = Math.max(0, 8 - Math.abs(hour - 13) * 2);
				const afternoon = Math.max(0, 11 - Math.abs(hour - 17) * 2);
				const dayBoost = weekday.value === 3 ? 4 : weekday.value === 5 ? -2 : 0;
				return {
					weekday: weekday.value,
					hour,
					checkins: Math.max(0, Math.round((midday + afternoon + dayBoost) * weekdayFactor))
				};
			})
		);

		return {
			...source,
			today: {
				visitors: lastDay.visitors,
				checkins: lastDay.checkins,
				new_accounts: lastDay.new_accounts
			},
			week: {
				visitors: currentWeek.visitors,
				checkins: currentWeek.checkins,
				new_accounts: currentWeek.new_accounts
			},
			selected: {
				visitors: Math.max(...daily.map((day) => day.cumulative_visitors), 428),
				checkins: totalCheckins || 612,
				new_accounts: totalAccounts || 37
			},
			daily,
			weekly,
			heatmap,
			coverage: {
				identified_checkins: totalCheckins,
				total_checkins: totalCheckins,
				identified_percent: 100,
				first_checkin: source.range.start
			}
		};
	}

	async function changeRange(): Promise<void> {
		loading = true;
		failure = '';
		try {
			liveActivity = await data.api.getActivity(selectedRange);
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to load activity.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Activity · mkr.cx</title></svelte:head>

<main class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-2 pb-12 md:px-6">
	<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p
				class="text-sm font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-400"
			>
				Makerspace pulse
			</p>
			<h1 class="mt-1 text-3xl font-bold text-gray-950 dark:text-white">Activity</h1>
			<p class="mt-1 text-gray-600 dark:text-gray-300">A quick look at visits and new accounts.</p>
		</div>
		<label
			class="flex w-full flex-col gap-1 text-sm font-medium text-gray-700 dark:text-gray-200 sm:w-56"
		>
			Time range
			<select
				bind:value={selectedRange}
				on:change={changeRange}
				disabled={loading}
				class="rounded-xl border border-gray-300 bg-white px-3 py-2.5 text-gray-950 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-white"
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
			class="rounded-xl border border-red-300 bg-red-50 p-4 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
		>
			{failure}
		</div>
	{/if}

	{#if previewing}
		<div
			class="flex flex-col gap-2 rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-100 sm:flex-row sm:items-center sm:justify-between"
		>
			<div>
				<strong>Sample-data preview</strong>
				<span class="ml-1">These illustrative numbers are not stored anywhere.</span>
			</div>
			<a class="font-semibold underline underline-offset-2" href="/staff/activity">Show live data</a
			>
		</div>
	{/if}

	<section aria-label="Activity summary" class="grid gap-3 sm:grid-cols-3">
		{#each [{ label: 'Today', value: activity.today }, { label: 'This week', value: activity.week }, { label: activity.range.label, value: activity.selected }] as card}
			<article
				class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-900"
			>
				<p class="text-sm font-medium text-gray-500 dark:text-gray-400">{card.label}</p>
				<div class="mt-2 flex items-baseline gap-2">
					<strong class="text-4xl font-bold tabular-nums text-gray-950 dark:text-white"
						>{card.value.visitors.toLocaleString()}</strong
					>
					<span class="text-sm text-gray-600 dark:text-gray-300">unique visitors</span>
				</div>
				<p
					class="mt-3 border-t border-gray-100 pt-3 text-sm text-gray-600 dark:border-gray-800 dark:text-gray-300"
				>
					<span class="font-semibold text-emerald-700 dark:text-emerald-400"
						>{card.value.new_accounts.toLocaleString()}</span
					>
					new {card.value.new_accounts === 1 ? 'account' : 'accounts'}
				</p>
			</article>
		{/each}
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-900 md:p-6"
		aria-labelledby="trend-heading"
	>
		<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
			<div>
				<h2 id="trend-heading" class="text-xl font-bold text-gray-950 dark:text-white">
					Visitor trend
				</h2>
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
					{shortDate(activity.range.start)}–{shortDate(activity.range.end)} · {activity.selected.checkins.toLocaleString()}
					total check-ins
				</p>
			</div>
			<div
				class="inline-flex w-fit rounded-lg bg-gray-100 p-1 dark:bg-gray-800"
				aria-label="Trend grouping"
			>
				{#each trendOptions as option}
					<button
						type="button"
						on:click={() => (trendMode = option.value)}
						class="rounded-md px-3 py-1.5 text-sm font-medium transition {trendMode === option.value
							? 'bg-white text-gray-950 shadow-sm dark:bg-gray-700 dark:text-white'
							: 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
						aria-pressed={trendMode === option.value}>{option.label}</button
					>
				{/each}
			</div>
		</div>

		{#if points.length > 0}
			<div class="mt-6 overflow-x-auto pb-2">
				<div
					class="flex h-56 min-w-[680px] items-end gap-1 border-b border-gray-200 px-1 dark:border-gray-700"
				>
					{#each points as point, index}
						<div
							class="group relative flex h-full min-w-[5px] flex-1 items-end"
							title={`${shortDate(point.start)}: ${point.value} ${trendMode === 'cumulative' ? 'visitors so far' : 'unique visitors'}`}
						>
							<div
								class="w-full rounded-t-sm bg-emerald-500 transition hover:bg-emerald-600 dark:bg-emerald-500 dark:hover:bg-emerald-400"
								style={`height: ${point.value === 0 ? 1 : Math.max(3, (point.value / chartMax) * 100)}%`}
							></div>
							{#if index === 0 || index === points.length - 1 || (trendMode === 'weekly' ? true : index % 14 === 0)}
								<span
									class="absolute -bottom-6 left-0 whitespace-nowrap text-[10px] text-gray-500 dark:text-gray-400"
									>{shortDate(point.start)}</span
								>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{:else}
			<div
				class="mt-6 rounded-xl bg-gray-50 p-6 text-center text-gray-500 dark:bg-gray-800 dark:text-gray-400"
			>
				<p>No activity recorded in this range yet.</p>
				<a
					class="mt-3 inline-flex rounded-lg bg-emerald-700 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-800"
					href="/staff/activity?preview=1">Preview with sample data</a
				>
			</div>
		{/if}
	</section>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-900 md:p-6"
		aria-labelledby="heat-heading"
	>
		<div>
			<h2 id="heat-heading" class="text-xl font-bold text-gray-950 dark:text-white">
				When people arrive
			</h2>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				Check-ins by weekday and hour. This shows arrivals, not occupancy.
			</p>
		</div>
		<div class="mt-5 overflow-x-auto pb-2">
			<div
				class="grid min-w-[720px] gap-1"
				style="grid-template-columns: 3rem repeat(15, minmax(2.25rem, 1fr));"
			>
				<div></div>
				{#each hours as hour}
					<div class="pb-1 text-center text-[10px] font-medium text-gray-500 dark:text-gray-400">
						{hourLabel(hour)}
					</div>
				{/each}
				{#each weekdays as weekday}
					<div class="flex items-center text-xs font-semibold text-gray-600 dark:text-gray-300">
						{weekday.label}
					</div>
					{#each hours as hour}
						{@const value = heatValue(weekday.value, hour)}
						<div
							class="flex aspect-square items-center justify-center rounded-md text-[10px] font-semibold {value /
								heatMax >
							0.45
								? 'text-white'
								: 'text-gray-700 dark:text-gray-200'}"
							style={heatStyle(value)}
							title={`${weekday.label} ${hourLabel(hour)}: ${value} check-ins`}
							aria-label={`${weekday.label} ${hourLabel(hour)}, ${value} check-ins`}
						>
							{value || ''}
						</div>
					{/each}
				{/each}
			</div>
		</div>
	</section>

	{#if activity.coverage.total_checkins > 0 && activity.coverage.identified_percent < 99.95}
		<p class="px-1 text-xs text-gray-500 dark:text-gray-400">
			{activity.coverage.identified_percent.toFixed(0)}% of check-ins in this range could be
			connected to a member. Unique visitor totals exclude the rest.
		</p>
	{/if}
</main>
