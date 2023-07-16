<script lang="ts">
    import type { Dayjs } from "dayjs";
    import { date_format } from "$lib/src/stores";

    export let time: Dayjs;
    let format: string = "MMM";

    date_format.subscribe((value) => {
        format = value;
    });

    $: timestamp = time.format(format);

</script>

{timestamp}