<script lang="ts">
	import Calendar from '$lib/components/Calendar.svelte';
</script>

<div class="flex w-full flex-col items-center px-2 pb-12 text-center md:px-8">
	<div class="flex w-full flex-col items-center justify-center">
		<Calendar url="/calendar" fitWorkingHours />
	</div>
</div>
