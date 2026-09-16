<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageData } from './$types';
	import PrinterHistory from '$lib/printers/PrinterHistory.svelte';
	import { duration, type Printer } from '$lib/printers/prototype-data';
	export let data: PageData;
	let printer: Printer | null;
	let canManage = false;
	let message = '';
	$: {
		printer = data.printer;
		canManage = data.canManage;
		message = '';
	}
	const conditions = {
		working: 'Working',
		limited: 'Limited use',
		out: 'Out of service',
		unknown: 'Condition unknown'
	};
	const lineups = {
		active: 'Active',
		testing: 'Testing',
		repair: 'In repair',
		shelved: 'Shelved',
		retired: 'Retired'
	};
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
			canManage = result.canManage;
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
			message = 'Live status unavailable.';
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
					{[printer.model, printer.machineId, printer.location].filter(Boolean).join(' · ')}
				</p>
			</div>
			{#if canManage}<a class="edit" href={`/printers/manage?id=${encodeURIComponent(printer.id)}`}
					>Edit</a
				>{/if}
		</header>
		<section class="summary" aria-label="Printer status">
			<div class="status">
				<strong class="condition {printer.condition}">{conditions[printer.condition]}</strong>
				<span
					>{printer.stale
						? 'Live status unavailable'
						: !printer.connected || printer.activity === 'unknown'
							? 'Offline'
							: printer.activity === 'printing'
								? 'Printing'
								: printer.activity === 'paused'
									? 'Paused'
									: 'Idle'}</span
				>
				{#if printer.lifecycle && printer.lifecycle !== 'active'}<span
						>{lineups[printer.lifecycle]}</span
					>{/if}
			</div>
			{#if printer.lastSeen && (!printer.connected || printer.stale)}<p class="last-seen">
					Last seen {date(printer.lastSeen)}
				</p>{/if}
			{#if printer.note}<p class="note">{printer.note}</p>{/if}
		</section>
		{#if !printer.stale && printer.job && ['printing', 'paused'].includes(printer.activity)}
			<section class="current" aria-label="Current print">
				<h2>Current print</h2>
				<p class="file">{printer.job.file}</p>
				<p>{[printer.job.person, printer.job.material].filter(Boolean).join(' · ')}</p>
				<div class="progress">
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
		{#key printer.id}<PrinterHistory id={printer.id} />{/key}
	{/if}
</main>

<style>
	main {
		max-width: 52rem;
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
		margin: 2rem 0 1.5rem;
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
	.edit {
		border: 1px solid #cdd1d6;
		border-radius: 0.4rem;
		padding: 0.45rem 1rem;
	}
	.summary {
		border: 1px solid #dfe2e6;
		border-radius: 0.6rem;
		padding: 1.1rem;
		background: #fafafa;
	}
	.status {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.6rem 1rem;
		font-size: 0.9rem;
	}
	.condition {
		border-radius: 0.3rem;
		padding: 0.2rem 0.5rem;
	}
	.working {
		color: #206441;
		background: #e5f3eb;
	}
	.limited {
		color: #79530a;
		background: #fff3d7;
	}
	.out {
		color: #971b25;
		background: #fde9ec;
	}
	.unknown {
		background: #e8eaed;
	}
	.last-seen {
		color: #66707c;
		font-size: 0.8rem;
		margin-top: 0.6rem;
	}
	.note {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		margin-top: 1rem;
		line-height: 1.55;
	}
	.current {
		margin: 1.5rem 0;
	}
	h2 {
		font-size: 1rem;
		font-weight: 650;
		margin-bottom: 0.6rem;
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
	:global(.dark) .unknown {
		color: #252b35;
	}
</style>
