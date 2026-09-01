<script lang="ts">
	import { Heading } from 'flowbite-svelte';

	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex flex-col items-center gap-6 text-center md:px-8">
	<Heading tag="h1" customSize="text-3xl font-extrabold md:text-4xl"
		>Welcome to the UMass Amherst Makerspace's Online Portal</Heading
	>
	<div class="flex w-full flex-col items-center justify-center">
		<Calendar url="/calendar" fitWorkingHours />
	</div>
</div>
