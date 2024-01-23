<script lang="ts">
    import { dateFormat, timeFormat, dateTimeFormat } from "$lib/stores";
	import { format, formatRelative } from "date-fns";

    export let timestamp: Date;
    export let textClass = "text-sm";
    export let formatter: "date" | "time" | "datetime" | "relative" = "datetime";
</script>

<time datetime={timestamp.toISOString()} class={textClass}>
    {#if formatter === "date"}
        {format(timestamp, $dateFormat)}
    {:else if formatter === "time"}
        {format(timestamp, $timeFormat)}
    {:else if formatter === "relative"}
        {formatRelative(timestamp, new Date())}
    {:else}
        {format(timestamp, $dateTimeFormat)}
    {/if}
</time>