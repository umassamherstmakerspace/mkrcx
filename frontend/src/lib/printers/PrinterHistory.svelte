<script lang="ts">
	import { onMount, tick } from 'svelte';
	import {
		historyItems,
		filterHistoryItems,
		printDuration,
		type HistoryFilter,
		type PrinterHistoryData
	} from './history-view';
	export let id: string;
	let history: PrinterHistoryData | null = null;
	let error = '';
	let loading = false;
	let mounted = false;
	let request: AbortController | null = null;
	let filter: HistoryFilter = 'updates';
	let results: HTMLOListElement;
	let minResultsHeight = 0;
	let pagesLoaded = false;
	let generation = 0;
	function selectFilter(next: HistoryFilter) {
		if (next === filter) return;
		minResultsHeight = Math.max(
			0,
			window.innerHeight - (results?.getBoundingClientRect().top ?? 0)
		);
		filter = next;
		void refresh();
	}
	$: items = history ? historyItems(history) : [];
	$: visible = filterHistoryItems(items, filter);
	const views: { id: HistoryFilter; label: string }[] = [
		{ id: 'all', label: 'All' },
		{ id: 'updates', label: 'Notes & errors' },
		{ id: 'prints', label: 'Prints' }
	];
	const date = (value: string) =>
		new Date(value).toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	const day = (value: string) =>
		new Date(`${value.slice(0, 10)}T12:00:00`).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	async function refresh(more = false, background = false) {
		if (!mounted || (more && (loading || !history?.nextCursor))) return;
		const current = ++generation;
		request?.abort();
		const controller = new AbortController();
		request = controller;
		loading = true;
		const previous = more ? history : null;
		if (!more && !background) {
			history = null;
			pagesLoaded = false;
		}
		const query = new URLSearchParams({ id, page: '1', filter });
		if (previous?.nextCursor) query.set('cursor', previous.nextCursor);
		const timeout = window.setTimeout(() => controller.abort(), 8_000);
		try {
			const response = await fetch(`/printers/history?${query}`, { signal: controller.signal });
			if (!response.ok)
				throw new Error(
					[401, 403].includes(response.status) ? 'Staff access required.' : 'History unavailable.'
				);
			const result: PrinterHistoryData = await response.json();
			if (!mounted || current !== generation) return;
			if (previous) {
				const scrollTop = window.scrollY;
				history = {
					...result,
					events: [...(previous.events ?? []), ...(result.events ?? [])],
					historical: [...(previous.historical ?? []), ...(result.historical ?? [])],
					summaries: [...(previous.summaries ?? []), ...(result.summaries ?? [])],
					edits: [
						...new Map(
							[...(previous.edits ?? []), ...(result.edits ?? [])].map((edit) => [
								edit.version,
								edit
							])
						).values()
					],
					pageIds: [...(previous.pageIds ?? []), ...(result.pageIds ?? [])]
				};
				pagesLoaded = true;
				await tick();
				if (mounted && current === generation) window.scrollTo({ top: scrollTop });
			} else history = result;
			error = '';
		} catch (e) {
			if (!mounted || current !== generation) return;
			error = e instanceof Error ? e.message : 'History unavailable.';
			if (error === 'Staff access required.') history = null;
		} finally {
			window.clearTimeout(timeout);
			if (current === generation) {
				loading = false;
				request = null;
			}
		}
	}
	onMount(() => {
		mounted = true;
		void refresh();
		const timer = window.setInterval(() => {
			if (!loading && !pagesLoaded) void refresh(false, true);
		}, 15_000);
		return () => {
			mounted = false;
			window.clearInterval(timer);
			request?.abort();
		};
	});
</script>

<section class="history" aria-label="Printer history">
	<div class="heading">
		<h2>History</h2>
		{#if pagesLoaded}<button type="button" disabled={loading} on:click={() => refresh()}
				>Refresh history</button
			>{/if}
	</div>
	{#if loading && !history && !error}<p role="status">Loading…</p>{/if}
	{#if error}<p role="status">{error}</p>{/if}
	{#if history}
		{#if history.usage && history.usage.jobs > 0}
			<p class="usage">
				<strong>Recorded print time: {printDuration(history.usage.seconds)}</strong> · {history
					.usage.jobs} prints recorded{#if history.usage.firstOutcome}
					{' '}since {new Date(history.usage.firstOutcome).toLocaleDateString(undefined, {
						year: 'numeric',
						month: 'short',
						day: 'numeric'
					})}{/if}
			</p>
			<p class="coverage">
				Partial history. Includes completed, cancelled and failed prints.{#if history.usage.missingDurations}
					{history.usage.missingDurations} missing durations.{/if}
			</p>
		{/if}
		{#each history.origins ?? [] as origin}<p class="coverage">{origin.body}</p>{/each}
		{#if history.legacy?.jobs}
			<p class="coverage">
				{history.legacy.jobs.toLocaleString()} earlier print submissions{#if history.legacy.first}
					{' '}since {day(history.legacy.first)}{/if}. Outcomes and actual durations were not
				recorded.
			</p>
		{/if}
		{#each history.meters ?? [] as meter}
			<p class="coverage">
				Historical meter: <strong>{meter.meterHours?.toLocaleString()} hours</strong>{' '}recorded {day(
					meter.recordedAt
				)}. Meter readings may reset; this is not a lifetime total.
			</p>
		{/each}
		<div class="history-filters" aria-label="History views">
			{#each views as view}<button
					class:active={filter === view.id}
					type="button"
					aria-pressed={filter === view.id}
					on:click={() => selectFilter(view.id)}>{view.label}</button
				>{/each}
		</div>
		<ol bind:this={results} style:min-height={`${minResultsHeight}px`}>
			{#each visible as item (item.id)}
				<li
					class={item.kind}
					class:completed={item.outcome === 'completed'}
					class:cancelled={item.outcome === 'cancelled'}
				>
					<span class="marker" aria-hidden="true">
						{#if item.kind === 'note' || item.kind === 'summary'}
							<svg
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="1.7"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M14 3H6a1 1 0 0 0-1 1v16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8l-5-5Z" />
								<path d="M14 3v5h5M8 12h8M8 16h6" />
							</svg>
						{:else}{item.icon ??
								(item.kind === 'error' ? '!' : item.kind === 'job' ? '✓' : '·')}{/if}
					</span>
					<article>
						<div class="entry-heading">
							<time datetime={item.dateOnly ? item.recordedAt.slice(0, 10) : item.recordedAt}
								>{item.dateOnly
									? `${day(item.recordedAt)} · Time not recorded`
									: date(item.recordedAt)}</time
							>
						</div>
						{#if item.kind === 'note'}
							<p class="text">
								Status note update{#if item.user}
									by {item.user.replace(' · Card tap', '')}{:else if item.automatic}
									· Automatic{/if}: {item.text ? `“${item.text}”` : 'Note cleared.'}
							</p>
						{:else}
							{#if item.title}<h3>{item.title}</h3>{/if}
							{#if item.user}<p class="entry-user">Reported by {item.user}</p>{/if}
						{/if}
						{#if item.file || item.person || item.material || item.duration}<p class="job-details">
								{[item.person || 'User not recorded', item.material, item.duration, item.file]
									.filter(Boolean)
									.join(' · ')}
							</p>{/if}
						{#if item.text && item.kind !== 'note'}<p class="text">
								{item.text}
							</p>{/if}
						{#each item.changes ?? [] as change}<p class="change-line">{change}</p>{/each}
						{#if item.preparedBy}<p class="sources">Summary by {item.preparedBy}</p>{/if}
					</article>
				</li>
			{:else}<li class="empty">
					{filter === 'updates'
						? 'No notes or errors recorded.'
						: filter === 'prints'
							? 'No prints recorded.'
							: 'No history yet.'}
				</li>{/each}
		</ol>
		{#if history.nextCursor}<button
				class="more"
				type="button"
				disabled={loading}
				on:click={() => refresh(true)}>{loading ? 'Loading…' : 'Load more'}</button
			>{/if}
	{/if}
</section>

<style>
	.usage {
		font-size: 0.85rem;
	}
	.coverage {
		font-size: 0.75rem;
		color: #66707c;
		margin-top: 0.15rem;
	}
	.history-filters {
		display: flex;
		gap: 0.4rem;
		margin: 0.7rem 0;
	}
	.history-filters button.active {
		background: #881c1c;
		color: white;
		border-color: #881c1c;
	}
	.sources {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem 0.8rem;
		font-size: 0.75rem;
		margin-top: 0.3rem;
		color: #66707c;
	}
	:global(.dark) .coverage,
	:global(.dark) .sources {
		color: #aab3c0;
	}
	.history {
		margin-top: 1.4rem;
		width: 100%;
	}
	.heading {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		margin-bottom: 0.8rem;
	}
	h2 {
		font-size: 1.2rem;
		font-weight: 650;
	}
	button {
		color: #881c1c;
		font-size: 0.85rem;
		border: 1px solid #d1d5db;
		border-radius: 0.35rem;
		padding: 0.35rem 0.7rem;
	}
	button:hover {
		background: #f5f5f5;
	}
	ol {
		overflow-anchor: none;
		list-style: none;
		padding: 0;
		margin: 0;
	}
	li {
		display: grid;
		grid-template-columns: 1.5rem minmax(0, 1fr);
		gap: 0.5rem;
		padding: 0.4rem 0;
		border-bottom: 1px solid #e5e7eb;
	}
	.marker {
		display: flex;
		justify-content: center;
		align-items: center;
		width: 1.4rem;
		height: 1.4rem;
		border-radius: 50%;
		background: #eef0f3;
		color: #66707c;
		font-size: 0.9rem;
	}
	.entry-heading {
		display: flex;
		flex-wrap: wrap;
		gap: 0.15rem 0.65rem;
		align-items: baseline;
	}
	h3 {
		font-size: 0.9rem;
		font-weight: 500;
	}
	time {
		color: #66707c;
		font-size: 0.8rem;
		font-weight: 650;
	}
	.entry-user {
		font-size: 0.8rem;
		margin-top: 0.15rem;
	}
	.job-details {
		font-size: 0.8rem;
		color: #505966;
		overflow-wrap: anywhere;
	}
	.text,
	.change-line {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		font-size: 0.9rem;
		line-height: 1.35;
		margin-top: 0.15rem;
	}
	.completed .marker {
		background: #e4f4e9;
		color: #15723a;
		font-weight: 700;
	}
	.cancelled .marker {
		background: #fde9ec;
		color: #a11d2d;
		font-weight: 700;
	}
	.error .marker {
		background: #fde9ec;
		color: #971b25;
		font-weight: 700;
	}
	.error .text {
		border-left: 2px solid #c85c64;
		padding: 0.5rem 0.75rem;
		background: #fff6f6;
	}
	.more {
		display: block;
		margin: 1rem auto 0;
	}
	.empty {
		display: block;
		color: #66707c;
	}
	:global(.dark) button {
		color: #f0a0a0;
		border-color: #596271;
	}
	:global(.dark) button:hover {
		background: #303845;
	}
	:global(.dark) li {
		border-color: #424b58;
	}
	:global(.dark) time,
	:global(.dark) .job-details,
	:global(.dark) .empty {
		color: #aab3c0;
	}
	:global(.dark) .error .text {
		background: #39282b;
	}
</style>
