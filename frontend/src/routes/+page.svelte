<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex w-full flex-col items-center px-2 pb-12 text-center md:px-8">
	<nav
		aria-label="Makerspace tools"
		class="mb-5 grid w-full max-w-4xl grid-cols-1 gap-3 sm:grid-cols-2"
	>
		<a
			href="/note"
			class="flex items-center justify-between gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-left font-semibold text-gray-950 transition hover:border-blue-400 focus:outline-none focus:ring-4 focus:ring-blue-200 dark:border-blue-900 dark:bg-blue-950 dark:text-white dark:focus:ring-blue-900"
		>
			<span>Send us a note</span><span
				aria-hidden="true"
				class="text-xl text-blue-800 dark:text-blue-200">→</span
			>
		</a>
		<a
			href="/printers"
			class="flex items-center justify-between gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-left font-semibold text-gray-950 transition hover:border-emerald-400 focus:outline-none focus:ring-4 focus:ring-emerald-200 dark:border-emerald-900 dark:bg-emerald-950 dark:text-white dark:focus:ring-emerald-900"
		>
			<span>3D Printer Status</span><span
				aria-hidden="true"
				class="text-xl text-emerald-800 dark:text-emerald-200">→</span
			>
		</a>
	</nav>
	<div class="flex w-full flex-col items-center justify-center">
		<Calendar url="/calendar" fitWorkingHours />
	</div>
</div>
