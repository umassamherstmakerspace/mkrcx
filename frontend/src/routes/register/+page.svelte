<script lang="ts">
	import { base } from '$app/paths';
	import { registrationAccountChooserUrl } from '$lib/registration';
</script>

<svelte:head>
	<title>Welcome to the Makerspace · mkr.cx</title>
</svelte:head>

<main class="mx-auto w-full max-w-lg py-6 sm:py-12">
	<h1 class="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
		Welcome to the Makerspace
	</h1>

	<section aria-labelledby="returning-member" class="mt-8">
		<h2 id="returning-member" class="text-xl font-semibold text-gray-900 dark:text-white">
			Already registered?
		</h2>
		<p class="mt-2 text-gray-600 dark:text-gray-300">You only need to register once.</p>
		<a
			href={`${base}/login?return_to=${encodeURIComponent(base || '/')}`}
			class="mt-4 inline-flex min-h-12 w-full items-center justify-center rounded-lg bg-blue-700 px-5 py-3 font-semibold text-white hover:bg-blue-800 focus:outline-none focus:ring-4 focus:ring-blue-300 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800"
		>
			Log in
		</a>
	</section>

	<section
		aria-labelledby="new-member"
		class="mt-8 border-t border-gray-200 pt-8 dark:border-gray-700"
	>
		<h2 id="new-member" class="text-xl font-semibold text-gray-900 dark:text-white">New here?</h2>
		<a
			href={registrationAccountChooserUrl()}
			class="mt-4 inline-flex min-h-12 w-full items-center justify-center rounded-lg border border-gray-300 bg-white px-5 py-3 text-center font-semibold text-gray-900 hover:bg-gray-100 focus:outline-none focus:ring-4 focus:ring-gray-200 dark:border-gray-600 dark:bg-gray-800 dark:text-white dark:hover:bg-gray-700 dark:focus:ring-gray-700"
		>
			Register for the first time
		</a>
	</section>

	<a
		href={base || '/'}
		class="mt-8 inline-block rounded text-blue-700 underline underline-offset-4 hover:text-blue-900 focus:outline-none focus:ring-4 focus:ring-blue-200 dark:text-blue-400 dark:hover:text-blue-300 dark:focus:ring-blue-800"
	>
		Go to the homepage
	</a>
</main>
