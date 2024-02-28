<script lang="ts">
	import { Heading, P } from "flowbite-svelte";

	import Map from "$lib/components/Map.svelte";
	import Calendar from "$lib/components/Calendar.svelte";
</script>

<div class="text-center flex flex-col items-center justify-center">
	<Heading tag="h1" class="mb-4" customSize="text-4xl font-extrabold  md:text-5xl lg:text-6xl">Welcome to the UMass Amherst Makerspace's Online Portal</Heading>
	<P class="mb-6 text-lg lg:text-xl sm:px-16 xl:px-48 dark:text-gray-400">Access all of our online services here.</P>
	<div class="w-full flex flex-col items-center justify-center">
		<P class="mb-6 text-lg lg:text-xl sm:px-16 xl:px-48 dark:text-gray-400">Here is our hours.</P>
		<Calendar />
	</div>
	<div class="w-full flex flex-col items-center justify-center">
		<P class="mb-6 text-lg lg:text-xl sm:px-16 xl:px-48 dark:text-gray-400">Find us on the map below.</P>
		<Map />
	</div>
</div>
