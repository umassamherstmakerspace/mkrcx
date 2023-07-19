<script lang="ts">
	import type { Dayjs } from 'dayjs';
	import { date_format } from '$lib/src/stores';
	import { DEFAULT_DATE_FORMAT } from '$lib/src/defaults';

	export let time: Dayjs;
	let format: string = DEFAULT_DATE_FORMAT;

	date_format.subscribe((value) => {
		format = value;
	});

	$: timestamp = time.format(format);
</script>

{timestamp}
