<script lang="ts">
	import { getDateLocale, getDateTimeFormat, getTheme } from '$lib/stores';
	import { format } from 'date-fns';

	import { Label, Select, P } from 'flowbite-svelte';

	const theme = getTheme();
	const dateLocale = getDateLocale();
	const dateTimeFormat = getDateTimeFormat();
</script>

<li class="flex flex-col space-y-10">
	<div class="flex flex-col justify-between">
		<Label for="theme-selector" class="mb-2 block">Theme</Label>
		<Select bind:value={$theme} id="theme-selector">
			<option value="dark">Dark</option>
			<option value="light">Light</option>
			<option value="system">System</option>
		</Select>
	</div>
	<div class="flex flex-col justify-between">
		<Label for="locale-selector" class="mb-2 block">Date locale</Label>
		<Select bind:value={$dateLocale} id="locale-selector">
			<option value="ISO">ISO 8601</option>
			<option value="US">American</option>
			<option value="EU">European</option>
		</Select>

		<P class="text-gray-500 dark:text-gray-400"
			>Example: <code>{$dateLocale} {format(new Date(), $dateTimeFormat)}</code></P
		>
	</div>
</li>
