<script lang="ts">
	import { onMount } from 'svelte';
	import PrinterHistory from '$lib/printers/PrinterHistory.svelte';
	type Record = {
		id: string;
		name: string;
		model: string;
		machineId: string;
		location: string;
		lifecycle: string;
		host: string;
		mac: string;
		condition: string;
		note: string;
		manual: boolean;
		version: number;
	};
	let records: Record[] = [];
	let edit: Record | null = null;
	let message = '';
	let ready = false;
	let saving = false;
	let isNew = false;
	async function load() {
		const response = await fetch('/printers/manage');
		if (!response.ok) {
			ready = false;
			throw new Error(
				response.status === 403
					? 'Printer management is currently limited to administrators and specifically authorized accounts.'
					: 'Sign in with an authorized account to manage printers.'
			);
		}
		records = await response.json();
		ready = true;
	}
	onMount(() => {
		void load().catch((e) => (message = e.message));
	});
	function select(record?: Record) {
		isNew = !record;
		edit = record
			? { ...record }
			: {
					id: '',
					name: '',
					model: 'K1',
					machineId: '',
					location: '',
					lifecycle: 'testing',
					host: '',
					mac: '',
					condition: 'unknown',
					note: '',
					manual: true,
					version: 0
				};
		message = '';
	}
	async function save() {
		if (!edit) return;
		saving = true;
		message = '';
		try {
			const {
				id,
				name,
				model,
				machineId,
				location,
				lifecycle,
				host,
				mac,
				condition,
				note,
				manual,
				version
			} = edit;
			const response = await fetch(`/printers/manage?id=${encodeURIComponent(id)}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name,
					model,
					machineId,
					location,
					lifecycle,
					host,
					mac,
					condition,
					note,
					manual,
					version
				})
			});
			if (!response.ok)
				throw new Error(
					response.status === 409
						? 'This record changed or its hardware identity conflicts. Reload the list and review before saving.'
						: await response.text()
				);
			edit = null;
			await load();
			message = 'Saved. The dashboard record is updated; local printer gating is unchanged.';
		} catch (e) {
			message = e instanceof Error ? e.message : 'Save failed';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head
	><title>Manage printers · UMass Makerspace</title><meta
		name="robots"
		content="noindex, nofollow"
	/></svelte:head
>
<main>
	<a href="/printers">← Printer status</a>
	<h1>Manage printers</h1>
	<p>
		Keep the lineup, testing and repair notes up to date. Notes are public. These records do not
		change local print gating.
	</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if ready}
		<button on:click={() => select()} disabled={saving}>Add printer</button>
		<button
			on:click={() =>
				load()
					.then(() => {
						edit = null;
						message = 'List reloaded.';
					})
					.catch((e) => (message = e.message))}
			disabled={saving}>Reload list</button
		>
		{#if edit}
			<form on:submit|preventDefault={save}>
				<h2>{isNew ? 'Add printer' : edit.name}</h2>
				<div class="fields">
					<label
						>Permanent ID<input
							required
							pattern={'[a-z0-9][a-z0-9-]{0,79}'}
							maxlength="80"
							disabled={!isNew || saving}
							bind:value={edit.id}
						/></label
					>
					<label
						>Name<input required maxlength="120" disabled={saving} bind:value={edit.name} /></label
					>
					<label
						>Model<input required maxlength="80" disabled={saving} bind:value={edit.model} /></label
					>
					<label
						>Machine label<input
							maxlength="120"
							disabled={saving}
							bind:value={edit.machineId}
						/></label
					>
					<label
						>Location<input maxlength="120" disabled={saving} bind:value={edit.location} /></label
					>
					<label
						>Lineup<select disabled={saving} bind:value={edit.lifecycle}
							><option value="active">Active lineup</option><option value="testing">Testing</option
							><option value="repair">In repair</option><option value="shelved"
								>Shelved — hidden from public list</option
							><option value="retired">Retired — hidden from public list</option></select
						></label
					>
					<label
						>Known condition<select
							disabled={saving}
							bind:value={edit.condition}
							on:change={() => {
								if (edit) edit.manual = true;
							}}
							><option value="working">Working</option><option value="limited">Limited use</option
							><option value="out">Out of service</option><option value="unknown"
								>Not yet known</option
							></select
						></label
					>
				</div>
				<label
					>Public note<textarea
						maxlength="2000"
						rows="4"
						disabled={saving}
						bind:value={edit.note}
						on:input={() => {
							if (edit) edit.manual = true;
						}}
					></textarea></label
				>
				<label class="check"
					><input type="checkbox" disabled={saving} bind:checked={edit.manual} />Keep this condition
					and note until someone edits them here.</label
				>
				<p class="hint">
					Uncheck to use the station's last reported condition and note. Editing a condition keeps
					the note unless you explicitly change or clear it.
				</p>
				<details>
					<summary>Printer connection</summary>
					<p>
						Leave both fields blank for an offline printer awaiting setup. A replacement physical
						machine needs a new record.
					</p>
					<div class="fields">
						<label>LAN IPv4 address<input disabled={saving} bind:value={edit.host} /></label><label
							>Hardware MAC address<input
								disabled={saving || (!isNew && !!records.find((p) => p.id === edit?.id)?.mac)}
								bind:value={edit.mac}
							/></label
						>
					</div>
				</details>
				<button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save record'}</button><button
					type="button"
					disabled={saving}
					on:click={() => (edit = null)}>Cancel</button
				>
			</form>
			{#if !isNew}{#key edit.id}<PrinterHistory id={edit.id} />{/key}{/if}
		{/if}
		<ul>
			{#each records as record}<li>
					<button disabled={saving} on:click={() => select(record)}>{record.name}</button><span
						>{record.model} · {record.lifecycle}</span
					>{#if record.note}<p>{record.note}</p>{/if}
				</li>{/each}
		</ul>
	{/if}
</main>

<style>
	main {
		max-width: 64rem;
		margin: 2rem auto;
		padding: 1rem;
	}
	h1 {
		font-size: 1.8rem;
	}
	h2 {
		font-size: 1.3rem;
	}
	p,
	h1,
	h2 {
		margin-bottom: 1rem;
	}
	button {
		padding: 0.55rem 0.8rem;
		margin: 0.35rem 0.5rem 0.35rem 0;
		border: 1px solid #aaa;
		border-radius: 0.35rem;
	}
	button:disabled {
		opacity: 0.5;
	}
	form {
		margin: 1rem 0;
		padding: 1rem;
		border: 1px solid #aaa;
		border-radius: 0.5rem;
	}
	.fields {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
		gap: 1rem;
	}
	label {
		display: grid;
		gap: 0.4rem;
		margin-bottom: 1rem;
	}
	input,
	select,
	textarea {
		color: #111;
		background: white;
		width: 100%;
		border: 1px solid #999;
		border-radius: 0.25rem;
		padding: 0.5rem;
	}
	.check {
		display: flex;
		align-items: center;
	}
	.check input {
		width: auto;
	}
	.hint {
		font-size: 0.85rem;
	}
	li {
		border-bottom: 1px solid #aaa;
		padding: 0.5rem 0;
	}
	li span {
		font-size: 0.85rem;
	}
	li p {
		white-space: pre-wrap;
	}
	details {
		margin: 1rem 0;
	}
</style>
