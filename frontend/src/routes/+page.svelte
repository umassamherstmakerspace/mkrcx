<script lang="ts">
	import { Heading, P } from 'flowbite-svelte';

	import Map from '$lib/components/Map.svelte';
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex flex-col items-center justify-center gap-12 text-center md:px-16">
	<Heading tag="h1" class="mb-4" customSize="text-4xl font-extrabold  md:text-5xl lg:text-6xl"
		>Welcome to the UMass Amherst Makerspace's Online Portal</Heading
	>
	<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48"
		>Access all of our online services here.</P
	>
	<div class="flex w-full flex-col items-center justify-center">
		<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48">Here is our hours.</P>
		<Calendar url="/calendar" />
	</div>
	<div class="flex w-full flex-col items-center justify-center">
		<P class="mb-6 text-lg dark:text-gray-400 sm:px-16 lg:text-xl xl:px-48"
			>Find us on the map below.</P
		>
		<Map />
	</div>
</div>
