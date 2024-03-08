<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
	import type { EventInput } from '@fullcalendar/core/index.js';
	import { Heading, P } from 'flowbite-svelte';
	import ColorHash from 'color-hash';

	function colorizeEvent(event: EventInput) {
		if (event.title === undefined) return;

		event.backgroundColor = new ColorHash().hex(event.title);
	}
</script>

<div class="flex flex-col items-center justify-center gap-12 px-16 text-center">
	<Heading tag="h1" class="mb-4" customSize="text-4xl font-extrabold  md:text-5xl lg:text-6xl"
		>Welcome to the Staff Portal</Heading
	>
	<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48"
		>Access staff resources here.</P
	>
	<div class="flex w-full flex-col items-center justify-center">
		<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48">Staff hours.</P>
		<Calendar url="/staff/calendar" {colorizeEvent} />
	</div>
</div>
