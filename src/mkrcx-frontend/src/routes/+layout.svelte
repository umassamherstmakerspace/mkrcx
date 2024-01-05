<script lang="ts">
	import Header from '$lib/components/Header.svelte';
	import SideMenu from '$lib/components/SideMenu.svelte';
	import { isDark } from '$lib/stores';
	import '../app.pcss';

	isDark.subscribe((value) => {
		if (value) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	});

	let hideSidebar = true;
</script>

<Header bind:hideSidebar />
<SideMenu bind:hidden={hideSidebar} />

<slot />
