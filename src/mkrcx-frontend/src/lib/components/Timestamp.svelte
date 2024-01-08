<script lang="ts">
    import { dateTimeFormat } from "$lib/stores";
	import { format } from "date-fns";

    export let timestamp: Date;
    export let textClass = "text-sm";
</script>

<time datetime={timestamp.toISOString()} class={textClass}>
    {format(timestamp, $dateTimeFormat)}
</time>