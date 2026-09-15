<script lang="ts">
	import { onMount } from 'svelte';
	export let id: string;
	type Event = {
		sourceId: string;
		recordedAt: string;
		eventType: string;
		detail: string;
		file?: string;
		material?: string;
	};
	type Edit = {
		recordedAt: string;
		actor: string;
		version: number;
		condition: string;
		note: string;
		manual: boolean;
		lifecycle: string;
		name: string;
	};
	type History = {
		events: Event[] | null;
		edits: Edit[];
		lastSync: string | null;
		stationCondition: string;
		stationNote: string;
		stationReportedAt: string | null;
	};
	let history: History | null = null;
	let error = '';
	let loading = false;
	let tab = 'events';
	const labels: Record<string, string> = {
		started: 'Print started',
		printer_completed: 'Print completed',
		printer_cancelled: 'Print cancelled',
		printer_failed: 'Print failed',
		printer_error: 'Printer error',
		staff_runtime_changed: 'Condition or note changed at printer',
		printer_runtime_changed: 'Station condition changed'
	};
	const conditionLabels: Record<string, string> = {
		working: 'Working',
		limited: 'Limited use',
		out: 'Out of service',
		unknown: 'Unknown'
	};
	async function refresh() {
		loading = true;
		error = '';
		try {
			const response = await fetch(`/printers/history?id=${encodeURIComponent(id)}`);
			if (!response.ok)
				throw new Error(
					response.status === 401 || response.status === 403
						? 'Staff access is required to view history.'
						: 'History is temporarily unavailable.'
				);
			history = await response.json();
		} catch (e) {
			history = null;
			error = e instanceof Error ? e.message : 'History unavailable.';
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		void refresh();
	});
	function date(value: string) {
		return new Date(value).toLocaleString();
	}
</script>

<section class="history" aria-label="Printer history">
	<div class="heading">
		<h3>History</h3>
		<button type="button" disabled={loading} on:click={refresh}
			>{loading ? 'Loading…' : 'Refresh history'}</button
		>
	</div>
	{#if error}<p role="status">{error}</p>{/if}
	{#if history}
		{#if history.stationCondition}<p class="context">
				Station’s last saved condition: {conditionLabels[history.stationCondition] ??
					history.stationCondition}{history.stationNote ? ` · ${history.stationNote}` : ''}.
				Dashboard edits take effect here; print restrictions are managed at the printer.
			</p>{/if}
		<div class="tabs" role="group" aria-label="History type">
			<button type="button" aria-pressed={tab === 'events'} on:click={() => (tab = 'events')}
				>Jobs & printer events</button
			><button type="button" aria-pressed={tab === 'edits'} on:click={() => (tab = 'edits')}
				>Dashboard edits</button
			>
		</div>
		{#if tab === 'events'}
			<p class="context">
				{history.lastSync
					? `Station history last collected ${date(history.lastSync)}. Showing up to 100 recent events.`
					: 'Station history has not been collected yet.'}
			</p>
			<ol>
				{#each history.events ?? [] as event (event.sourceId)}<li>
						<strong>{labels[event.eventType] ?? event.eventType.replaceAll('_', ' ')}</strong><time
							>{date(event.recordedAt)}</time
						>{#if event.file}<p class="filename">
								{event.file}{event.material ? ` · ${event.material}` : ''}
							</p>{/if}{#if event.detail}<p class="detail">{event.detail}</p>{/if}
					</li>{:else}<li>No recorded events available for this printer.</li>{/each}
			</ol>
		{:else}
			<ol>
				{#each history.edits as edit (edit.version)}<li>
						<strong>{edit.name} · {edit.lifecycle}</strong><time
							>{date(edit.recordedAt)} · {edit.actor}</time
						>
						<p>
							{edit.manual
								? `Saved condition: ${conditionLabels[edit.condition] ?? edit.condition}`
								: 'Using the station’s saved condition and note'}
						</p>
						{#if edit.manual && edit.note}<p class="detail">{edit.note}</p>{/if}
					</li>{:else}<li>No dashboard edits recorded yet.</li>{/each}
			</ol>
		{/if}
	{/if}
</section>

<style>
	.history {
		margin-top: 1rem;
		border-top: 1px solid #d1d5db;
		padding-top: 1rem;
		width: 100%;
	}
	.heading,
	.tabs {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		flex-wrap: wrap;
	}
	.heading {
		justify-content: space-between;
	}
	h3 {
		font-size: 1.1rem;
		font-weight: 650;
	}
	button {
		border: 1px solid #9ca3af;
		border-radius: 0.35rem;
		padding: 0.35rem 0.6rem;
	}
	button[aria-pressed='true'] {
		background: #881c1c;
		border-color: #881c1c;
		color: white;
	}
	ol {
		list-style: none;
		padding: 0;
		max-height: 28rem;
		overflow: auto;
	}
	li {
		border-bottom: 1px solid #e5e7eb;
		padding: 0.75rem 0;
	}
	time {
		display: block;
		font-size: 0.8rem;
		opacity: 0.8;
	}
	.context {
		font-size: 0.85rem;
		margin: 0.7rem 0;
	}
	.detail {
		white-space: pre-wrap;
	}
	.detail,
	.filename {
		overflow-wrap: anywhere;
	}
</style>
