<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import SideMenu from '$lib/components/SideMenu.svelte';
	import { isDark } from '$lib/stores';
	import '../app.pcss';
	import type { LayoutData } from './$types';

	isDark.subscribe((value) => {
		if (value) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	});

	let hideSidebar = true;
	
	export let data: LayoutData;
	let { user } = data;
</script>

<Header bind:hideSidebar user={user} />
<SideMenu bind:hidden={hideSidebar} user={user} />

<div class="p-4">
	<slot />
</div>
