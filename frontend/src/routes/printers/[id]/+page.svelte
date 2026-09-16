<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageData } from './$types';
	import { activityState, conditionLabels } from '$lib/printers/printer-state';
	import FleetStamp from '$lib/printers/FleetStamp.svelte';
	import PrinterHistory from '$lib/printers/PrinterHistory.svelte';
	import { duration, type Printer } from '$lib/printers/prototype-data';
	export let data: PageData;
	let printer: Printer | null;
	let message = '';
	$: {
		printer = data.printer;
		message = '';
	}
	const date = (value: string) => new Date(value).toLocaleString();
	async function refresh() {
		const id = data.printer.id;
		try {
			const response = await fetch(`/printers/${encodeURIComponent(id)}`, {
				headers: { Accept: 'application/json' }
			});
			if (id !== data.printer.id) return;
			if ([401, 403, 404].includes(response.status)) {
				printer = null;
				message = response.status === 404 ? 'Printer not found.' : 'Staff sign-in required.';
				return;
			}
			if (!response.ok) throw new Error();
			const result = await response.json();
			if (id !== data.printer.id) return;
			printer = result.printer;
			message = '';
		} catch {
			if (id !== data.printer.id) return;
			if (printer)
				printer = {
					...printer,
					stale: true,
					job: undefined,
					minutes: undefined,
					progress: undefined
				};
			message = '';
		}
	}
	onMount(() => {
		const timer = window.setInterval(refresh, 15_000);
		return () => window.clearInterval(timer);
	});
</script>

<svelte:head>
	<title>{printer?.name ?? 'Printer'} · UMass Makerspace</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<main>
	<a class="back" href="/printers">← Printers</a>
	{#if message}<p role="status" class="message">{message}</p>{/if}
	{#if printer}
		<header>
			<div>
				<p class="eyebrow">Staff view</p>
				<h1>{printer.name}</h1>
				<p class="identity">
					{[printer.model, printer.machineId].filter(Boolean).join(' · ')}
				</p>
			</div>
		</header>
		<section class="summary" aria-label="Printer status">
			<div class="status-line">
				<strong class={printer.condition}
					>{printer.condition === 'unknown'
						? 'Condition unknown'
						: conditionLabels[printer.condition]}</strong
				>
				<FleetStamp lifecycle={printer.lifecycle} />
			</div>
			{#if printer.location}<p class="location">{printer.location}</p>{/if}
			{#if printer.nextAction}<p class="next-action">
					<strong>Next:</strong>
					{printer.nextAction}
				</p>{/if}
			{#if printer.fault && !printer.stale}
				<section class="machine-error">
					<h2>Printer error</h2>
					<p>{printer.fault}</p>
				</section>
			{:else if ['offline', 'unavailable'].includes(activityState(printer))}
				<p class="last-seen">
					{activityState(printer) === 'offline'
						? 'Offline'
						: 'Updates unavailable'}{#if printer.lastSeen}
						· Last seen {date(printer.lastSeen)}{/if}
				</p>
			{:else if activityState(printer) === 'idle'}
				<p class="idle">Idle</p>
			{/if}
			{#if ['printing', 'paused'].includes(activityState(printer))}
				<section class="current" aria-label="Current print">
					{#if printer.job?.file}<p class="file">{printer.job.file}</p>{/if}
					{#if printer.job}<p>
							Printing for {printer.job.person || 'Unavailable'}{#if printer.job.material}
								· {printer.job.material}{/if}
						</p>{/if}
					{#if printer.job?.started}<p class="started">
							Started {Number.isNaN(Date.parse(printer.job.started))
								? printer.job.started
								: date(printer.job.started)}
						</p>{/if}
					<div class="progress">
						{#if printer.activity === 'paused'}<strong>Paused</strong
							>{:else if !printer.job?.file && !printer.job?.person && !printer.job?.material && printer.progress === undefined && printer.minutes === undefined}<span
								>Printing</span
							>{/if}
						{#if printer.progress !== undefined}<progress
								max="100"
								value={printer.progress}
								aria-label="Print progress"
							></progress><span>{Math.round(printer.progress)}%</span>{/if}
						{#if printer.minutes !== undefined}<span
								>~{duration(Math.max(0, Math.ceil(printer.minutes)))} left</span
							>{/if}
					</div>
				</section>
			{/if}
			{#if printer.note}<section class="notes" aria-label="Printer notes">
					<h2>Notes</h2>
					<p class="note">{printer.note}</p>
				</section>{/if}
			{#if printer.printerNoteAt}<section class="notes" aria-label="Latest printer note">
					<h2>Latest printer note <small>{date(printer.printerNoteAt)}</small></h2>
					<p class="note">{printer.printerNote || 'Note cleared at the printer.'}</p>
				</section>{/if}
		</section>
		{#key printer.id}<PrinterHistory id={printer.id} />{/key}
	{/if}
</main>

<style>
	.started {
		font-size: 0.8rem;
		color: #66707c;
	}
	:global(.dark) .started {
		color: #aab3c0;
	}
	.machine-error {
		margin-top: 0.7rem;
		border-left: 3px solid #a12a37;
		padding: 0.5rem 0.75rem;
	}
	.machine-error h2 {
		font-size: 0.9rem;
		font-weight: 650;
	}
	.machine-error p {
		font-size: 0.85rem;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.notes small {
		font-weight: 400;
		font-size: 0.75rem;
		margin-left: 0.4rem;
	}
	main {
		max-width: 1600px;
		margin: 0 auto;
		padding: 2rem 1.25rem 4rem;
		color: #252b35;
	}
	a {
		color: #881c1c;
	}
	a:hover {
		text-decoration: underline;
	}
	.back {
		font-size: 0.9rem;
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		margin: 1.25rem 0 1rem;
	}
	header > div {
		min-width: 0;
	}
	h1 {
		font-size: 2rem;
		font-weight: 650;
		line-height: 1.2;
		overflow-wrap: anywhere;
	}
	.eyebrow {
		color: #66707c;
		font-size: 0.75rem;
		margin-bottom: 0.35rem;
	}
	.identity {
		color: #66707c;
		margin-top: 0.4rem;
		font-size: 0.9rem;
		overflow-wrap: anywhere;
	}
	.summary {
		border: 1px solid #dfe2e6;
		border-radius: 0.6rem;
		padding: 1.1rem;
		background: #fafafa;
	}
	.status-line {
		display: flex;
		flex-wrap: wrap;
		gap: 0.7rem;
		align-items: center;
	}
	.status-line strong {
		font-size: 1rem;
		font-weight: 600;
	}
	.location,
	.next-action {
		margin-top: 0.4rem;
		font-size: 0.9rem;
	}
	.idle {
		margin-top: 0.4rem;
		font-size: 0.9rem;
	}
	.working {
		color: #206441;
	}
	.limited {
		color: #79530a;
	}
	.out {
		color: #971b25;
	}

	.last-seen {
		color: #66707c;
		font-size: 0.8rem;
		margin-top: 0.6rem;
	}
	.note {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		line-height: 1.55;
	}
	.notes {
		margin-top: 0.8rem;
	}
	.current {
		margin: 0.75rem 0;
	}
	h2 {
		font-size: 0.85rem;
		font-weight: 650;
		margin-bottom: 0.25rem;
	}
	.file {
		overflow-wrap: anywhere;
	}
	.progress {
		display: flex;
		flex-wrap: wrap;
		gap: 0.6rem;
		align-items: center;
		font-size: 0.85rem;
		margin-top: 0.5rem;
	}
	progress {
		width: 8rem;
		height: 0.45rem;
		accent-color: #881c1c;
	}
	.message {
		margin-top: 1rem;
	}
	:global(.dark) main {
		color: #e2e6eb;
	}
	:global(.dark) a {
		color: #f0a0a0;
	}
	:global(.dark) .summary {
		background: #202731;
		border-color: #424b58;
	}
	:global(.dark) .eyebrow,
	:global(.dark) .identity,
	:global(.dark) .last-seen {
		color: #aab3c0;
	}
	:global(.dark) .working {
		color: #8ed3aa;
	}
	:global(.dark) .limited {
		color: #e4c680;
	}
	:global(.dark) .out {
		color: #f2a1a8;
	}
</style>
