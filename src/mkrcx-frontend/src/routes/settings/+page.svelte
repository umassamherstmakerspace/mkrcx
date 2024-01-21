<script lang="ts">
    import { theme, dateLocale, dateTimeFormat } from "$lib/stores";
	import { format } from "date-fns";

    import { Label, Select, P } from "flowbite-svelte";
</script>

<li class="py-4 px-4 flex flex-col space-y-10">
    <div class="flex justify-between flex-col">
        <Label for="large-input" class="mb-2 block">Theme</Label>
        <Select bind:value={$theme}>
            <option value="dark">Dark</option>
            <option value="light">Light</option>
            <option value="system">System</option>
        </Select>
    </div>
    <div class="flex justify-between flex-col">
    <Label for="large-input" class="mb-2 block">Date locale</Label>
    <Select bind:value={$dateLocale}>
        <option value="ISO">ISO 8601</option>
        <option value="US">American</option>
        <option value="EU">European</option>
    </Select>

    <P class="text-gray-500 dark:text-gray-400">Example: <code>{$dateLocale} {format(new Date(), $dateTimeFormat)}</code></P>
</div>
</li>


