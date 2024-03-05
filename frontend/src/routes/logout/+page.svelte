<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import type { PageData } from './$types';

	export let data: PageData;
	const { user, api } = data;

	afterNavigate(() => {
		if (user) {
			window.location.href = api.logout($page.url.href);
		} else {
			window.location.href = '/';
		}
	});
</script>
