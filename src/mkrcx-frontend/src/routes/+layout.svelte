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

<svelte:head>
  <link rel="icon" type="image/svg" href='/favicon.svg' />
</svelte:head>


<SideMenu bind:hidden={hideSidebar} {user} />
<div class="flex h-dvh w-dvw flex-col">
	<Header bind:hideSidebar {user} />
	<div class="h-full w-full flex-1 p-4 max-h-full overflow-auto">
		<slot />
	</div>
</div>
