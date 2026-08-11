<script lang="ts">
	import type { PageData } from './$types';

	export let data: PageData;

	function dateInputValue(date: Date): string {
		const year = date.getFullYear();
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	const today = new Date();
	const thirtyDaysAgo = new Date(today);
	thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 29);

	let startDate = dateInputValue(thirtyDaysAgo);
	let endDate = dateInputValue(today);
	let downloading = false;
	let failure = '';

	async function downloadCSV(): Promise<void> {
		failure = '';
		const start = new Date(`${startDate}T00:00:00`);
		const endExclusive = new Date(`${endDate}T00:00:00`);
		endExclusive.setDate(endExclusive.getDate() + 1);

		if (Number.isNaN(start.valueOf()) || Number.isNaN(endExclusive.valueOf())) {
			failure = 'Choose a valid start and end date.';
			return;
		}
		if (start >= endExclusive) {
			failure = 'The start date must be on or before the end date.';
			return;
		}

		downloading = true;
		try {
			const { blob, filename } = await data.api.downloadCheckinCSV(start, endExclusive);
			const objectURL = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = objectURL;
			link.download = filename;
			link.click();
			URL.revokeObjectURL(objectURL);
		} catch (error) {
			failure = error instanceof Error ? error.message : 'Unable to export check-in data.';
		} finally {
			downloading = false;
		}
	}
</script>

<svelte:head><title>Check-in data · mkr.cx</title></svelte:head>

<main class="mx-auto flex max-w-3xl flex-col gap-6">
	<header>
		<p class="text-sm font-semibold uppercase tracking-[0.18em] text-blue-600 dark:text-blue-400">
			Staff zone
		</p>
		<h1 class="text-3xl font-bold text-gray-950 dark:text-white">Check-in data</h1>
		<p class="mt-2 text-gray-600 dark:text-gray-300">
			Download card-tap events as CSV. Member UUIDs remain stable between exports and do not reveal
			a member's name, account ID, or card number.
		</p>
	</header>

	<section
		class="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-900"
	>
		<form class="flex flex-col gap-5" on:submit|preventDefault={downloadCSV}>
			<div class="grid gap-4 sm:grid-cols-2">
				<label class="flex flex-col gap-2 font-medium text-gray-900 dark:text-white">
					Start date
					<input
						type="date"
						bind:value={startDate}
						required
						class="rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
					/>
				</label>
				<label class="flex flex-col gap-2 font-medium text-gray-900 dark:text-white">
					End date
					<input
						type="date"
						bind:value={endDate}
						required
						class="rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
					/>
				</label>
			</div>

			<p class="text-sm text-gray-500 dark:text-gray-400">
				Dates use this device's local timezone. The selected end date is included. Exports are
				limited to 370 days and 250,000 rows.
			</p>

			{#if failure}
				<div
					role="alert"
					class="rounded-xl border border-red-300 bg-red-50 p-4 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
				>
					{failure}
				</div>
			{/if}

			<button
				type="submit"
				disabled={downloading}
				class="inline-flex w-fit items-center rounded-lg bg-blue-700 px-5 py-2.5 font-medium text-white hover:bg-blue-800 focus:outline-none focus:ring-4 focus:ring-blue-300 disabled:cursor-wait disabled:opacity-60 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800"
			>
				{downloading ? 'Preparing CSV…' : 'Download CSV'}
			</button>
		</form>
	</section>

	<section
		class="rounded-xl border border-gray-200 p-5 text-sm text-gray-600 dark:border-gray-700 dark:text-gray-300"
	>
		<h2 class="font-semibold text-gray-950 dark:text-white">CSV fields</h2>
		<p class="mt-2">
			Event ID, UTC timestamp, stable member UUID, linked-at-tap status, identity resolution,
			outcome, and source. Unlinked cards have a blank member UUID unless a member links the card
			later.
		</p>
	</section>
</main>
