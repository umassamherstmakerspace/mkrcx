<script lang="ts" context="module">
	import { browser } from '$app/environment';
	import { colorScheme, SvelteUIProvider, ActionIcon, type ColorScheme } from '@svelteuidev/core';
	import { Sun, Moon } from 'radix-icons-svelte';
	import Cookies from 'js-cookie';

	let theme: ColorScheme = 'light';

	if (browser) {
		theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';

		if (Cookies.get('color_scheme')) {
			theme = Cookies.get('color_scheme') as ColorScheme;
		}
	}

	colorScheme.set(theme);

	function toggleTheme() {
		Cookies.set('color_scheme', theme === 'light' ? 'dark' : 'light');
		colorScheme.update((v) => (theme = v === 'light' ? 'dark' : 'light'));
	}
</script>

<SvelteUIProvider withGlobalStyles themeObserver={$colorScheme}>
	<ActionIcon size={30} color="gray" variant="outline" on:click={toggleTheme}>
		{#if $colorScheme === 'light'}
			<Sun />
		{:else}
			<Moon />
		{/if}
	</ActionIcon>
</SvelteUIProvider>
