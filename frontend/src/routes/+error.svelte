<script lang="ts">
	import { Heading, P } from 'flowbite-svelte';
	import { page } from '$app/stores';
</script>

<div class="flex h-full w-full flex-col items-center justify-center space-y-10">
	<Heading level="1" size="xl" class="mb-4 text-center">Error {$page?.status}</Heading>
	<P>{$page?.error?.message}</P>
</div>
