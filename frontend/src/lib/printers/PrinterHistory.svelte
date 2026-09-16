<script lang="ts">
	import { onMount } from 'svelte';
	import { historyItems, printDuration, type PrinterHistoryData } from './history-view';
	export let id: string;
	let history: PrinterHistoryData | null = null;
	let error = '';
	let loading = false;
	let mounted = false;
	let request: AbortController | null = null;
	let shown = 10;
	let filter = 'all';
	let results: HTMLOListElement;
	let minResultsHeight = 0;
	function selectFilter(next: string) {
		if (next === filter) return;
		let scroller: HTMLElement | null = results.parentElement;
		while (scroller && !/(auto|scroll)/.test(getComputedStyle(scroller).overflowY)) {
			scroller = scroller.parentElement;
		}
		// A short result list must not collapse the page beneath the current viewport.
		const bottom = scroller?.getBoundingClientRect().bottom ?? window.innerHeight;
		minResultsHeight = Math.max(0, bottom - results.getBoundingClientRect().top);
		filter = next;
		shown = 10;
	}
	$: items = history ? historyItems(history) : [];
	$: visible = items.filter(
		(item) => filter === 'all' || (filter === 'prints' ? item.printOutcome : item.kind !== 'job')
	);
	const date = (value: string) =>
		new Date(value).toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	async function refresh() {
		if (loading || !mounted) return;
		loading = true;
		request = new AbortController();
		const timeout = window.setTimeout(() => request?.abort(), 8_000);
		try {
			const response = await fetch(`/printers/history?id=${encodeURIComponent(id)}`, {
				signal: request.signal
			});
			if (!response.ok)
				throw new Error(
					[401, 403].includes(response.status) ? 'Staff access required.' : 'History unavailable.'
				);
			const result = await response.json();
			if (!mounted) return;
			history = result;
			error = '';
		} catch (e) {
			if (!mounted) return;
			history = null;
			error = e instanceof Error ? e.message : 'History unavailable.';
		} finally {
			window.clearTimeout(timeout);
			request = null;
			loading = false;
		}
	}
	onMount(() => {
		mounted = true;
		void refresh();
		const timer = window.setInterval(refresh, 15_000);
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
		<div class="history-filters" aria-label="History views">
			{#each [{ id: 'all', label: 'All' }, { id: 'updates', label: 'Notes & errors' }, { id: 'prints', label: 'Prints' }] as view}<button
					class:active={filter === view.id}
					type="button"
					aria-pressed={filter === view.id}
					on:click={() => selectFilter(view.id)}>{view.label}</button
				>{/each}
		</div>
		<ol bind:this={results} style:min-height={`${minResultsHeight}px`}>
			{#each visible.slice(0, shown) as item (item.id)}
				<li class={item.kind}>
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
									? new Date(item.recordedAt).toLocaleDateString(undefined, {
											year: 'numeric',
											month: 'short',
											day: 'numeric'
										})
									: date(item.recordedAt)}</time
							>
							{#if item.source}<span class="source">{item.source}</span>{/if}
							<h3>{item.title}</h3>
						</div>
						{#if item.user}<p class="entry-user">User: {item.user}</p>{/if}
						{#if item.file || item.person || item.material || item.duration}<p class="job-details">
								{[item.person || 'User not recorded', item.material, item.duration, item.file]
									.filter(Boolean)
									.join(' · ')}
							</p>{/if}
						{#if item.text}<p class="text">
								{#if item.kind === 'note'}“{item.text}”{:else}{item.text}{/if}
							</p>{/if}
						{#each item.changes ?? [] as change}<p class="change-line">{change}</p>{/each}
						{#if item.links?.length || item.preparedBy}<p class="sources">
								{#each item.links ?? [] as link}<a
										href={link.url}
										target="_blank"
										rel="noopener noreferrer">{link.label}</a
									>{/each}{#if item.preparedBy}<span>Summary by {item.preparedBy}</span>{/if}
							</p>{/if}
					</article>
				</li>
			{:else}<li class="empty">No history yet.</li>{/each}
		</ol>
		{#if visible.length > shown}<button class="more" type="button" on:click={() => (shown += 10)}
				>Load more</button
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
	.sources a {
		color: #881c1c;
		text-decoration: underline;
	}
	:global(.dark) .sources a {
		color: #f0a0a0;
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
	.source {
		color: #66707c;
		font-size: 0.75rem;
		font-weight: 650;
		overflow-wrap: anywhere;
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
	:global(.dark) .source,
	:global(.dark) time,
	:global(.dark) .job-details,
	:global(.dark) .empty {
		color: #aab3c0;
	}
	:global(.dark) .error .text {
		background: #39282b;
	}
</style>
