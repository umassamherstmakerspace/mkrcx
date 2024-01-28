<script lang="ts">
    import { Heading, P } from 'flowbite-svelte';
    import { page } from '$app/stores';
</script>

<div class="flex flex-col items-center justify-center w-full h-full space-y-10">
    <Heading level="1" size="xl" class="text-center mb-4">Error {$page?.status}</Heading>
    <P>{$page?.error?.message}</P>
</div>