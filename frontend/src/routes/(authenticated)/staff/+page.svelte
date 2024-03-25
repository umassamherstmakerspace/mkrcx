<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
	import { Heading, P } from 'flowbite-svelte';
</script>

<div class="flex flex-col items-center justify-center gap-12 text-center md:px-16">
	<Heading tag="h1" class="mb-4" customSize="text-4xl font-extrabold  md:text-5xl lg:text-6xl"
		>Welcome to the Staff Portal</Heading
	>
	<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48"
		>Access staff resources here.</P
	>
	<div class="flex w-full flex-col items-center justify-center">
		<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48">Staff hours.</P>
		<Calendar url="/staff/calendar" />
	</div>
</div>
