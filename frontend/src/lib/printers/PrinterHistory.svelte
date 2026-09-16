<script lang="ts">
	import { onMount } from 'svelte';
	import { historyItems, type PrinterHistoryData } from './history-view';
	export let id: string;
	let history: PrinterHistoryData | null = null;
	let error = '';
	let loading = false;
	let mounted = false;
	let request: AbortController | null = null;
	let shown = 10;
	$: items = history ? historyItems(history) : [];
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
		<ol>
			{#each items.slice(0, shown) as item (item.id)}
				<li class={item.kind}>
					<span class="marker" aria-hidden="true"
						>{item.icon ??
							(item.kind === 'note'
								? '✎'
								: item.kind === 'error'
									? '!'
									: item.kind === 'job'
										? '✓'
										: '·')}</span
					>
					<article>
						<div class="entry-heading">
							<time datetime={item.recordedAt}>{date(item.recordedAt)}</time>
							<h3>{item.title}</h3>
							{#if item.source}<span class="source" title={item.actor}>{item.source}</span>{/if}
						</div>
						{#if item.file || item.person || item.material || item.duration}<p class="job-details">
								{[item.file, item.person || 'User not recorded', item.material, item.duration]
									.filter(Boolean)
									.join(' · ')}
							</p>{/if}
						{#if item.text}<p class="text">{item.text}</p>{/if}
						{#each item.changes ?? [] as change}<p class="change-line">{change}</p>{/each}
					</article>
				</li>
			{:else}<li class="empty">No history yet.</li>{/each}
		</ol>
		{#if items.length > shown}<button class="more" type="button" on:click={() => (shown += 10)}
				>Load more</button
			>{/if}
	{/if}
</section>

<style>
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
		font-weight: 650;
	}
	time {
		color: #66707c;
		font-size: 0.8rem;
		font-weight: 500;
	}
	.source {
		color: #66707c;
		font-size: 0.75rem;
		overflow-wrap: anywhere;
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
	.note article {
		background: #f5f3ed;
		border-radius: 0.45rem;
		padding: 0.5rem 0.7rem;
	}
	.note .marker {
		background: #eee8d7;
		color: #786333;
		margin-top: 0.4rem;
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
	:global(.dark) .note article {
		background: #303027;
	}
	:global(.dark) .error .text {
		background: #39282b;
	}
	@media (max-width: 480px) {
		.note article {
			padding: 0.7rem;
		}
	}
</style>
