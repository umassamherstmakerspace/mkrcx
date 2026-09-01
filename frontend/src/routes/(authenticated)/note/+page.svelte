<script lang="ts">
	import type { ActionData, PageData } from './$types';

	export let data: PageData;
	export let form: ActionData;
	let note = form?.note ?? '';
	$: if (form?.success) note = '';
</script>

<svelte:head>
	<title>Send a note · UMass Makerspace</title>
</svelte:head>

<div class="mx-auto w-full max-w-2xl px-2 pb-12 pt-2 md:px-6 md:pt-6">
	<header>
		<h1 class="text-3xl font-bold text-gray-950 dark:text-white">Send a note</h1>
		<p class="mt-2 text-gray-700 dark:text-gray-200">
			Share an observation with the Makerspace team.
		</p>
	</header>

	{#if form?.success}
		<div
			class="mt-5 rounded-lg border border-green-300 bg-green-50 p-4 font-medium text-green-900 dark:border-green-800 dark:bg-green-950 dark:text-green-100"
			role="status"
		>
			Thanks! Your note has been submitted.
		</div>
	{:else if form?.message}
		<div
			class="mt-5 rounded-lg border border-red-300 bg-red-50 p-4 font-medium text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-100"
			role="alert"
		>
			{form.message}
		</div>
	{/if}

	<form method="POST" class="mt-6 space-y-4">
		<input type="hidden" name="submission_id" value={form?.submissionId ?? data.submissionId} />
		<div>
			<label for="note" class="mb-2 block text-lg font-semibold text-gray-950 dark:text-white"
				>Your note</label
			>
			<textarea
				id="note"
				name="note"
				bind:value={note}
				required
				maxlength="1500"
				rows="8"
				class="block w-full rounded-lg border border-gray-300 bg-white p-3 text-base text-gray-950 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
				placeholder="What should we know?"
			></textarea>
			<p class="mt-1 text-right text-sm text-gray-500 dark:text-gray-400">
				{[...note].length} / 1500
			</p>
		</div>

		<div
			class="rounded-lg bg-gray-100 p-4 text-sm text-gray-700 dark:bg-gray-800 dark:text-gray-200"
		>
			<p>The note will be shared with Makerspace staff along with your name and email.</p>
			<p class="mt-2">
				Need to include photos or attachments? Send those to
				<a class="font-medium underline" href="mailto:makerspace@umass.edu">makerspace@umass.edu</a
				>. Thanks!
			</p>
		</div>

		<button
			type="submit"
			class="inline-flex min-h-12 w-full items-center justify-center rounded-lg bg-blue-700 px-5 py-3 text-lg font-semibold text-white transition hover:bg-blue-800 focus:outline-none focus:ring-4 focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800 sm:w-auto"
		>
			Send note
		</button>
	</form>
</div>
