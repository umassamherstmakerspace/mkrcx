<script lang="ts">
	import HeadContent from '$lib/components/HeadContent.svelte';
	import { Header, SvelteUIProvider } from '@svelteuidev/core';
	import { quadIn, quadOut } from 'svelte/easing';
	import { slide } from 'svelte/transition';
	import { colorScheme } from '@svelteuidev/core';
	import Cookies from 'js-cookie';
	import { DEFAULT_THEME } from '$lib/src/defaults';
	import { theme, screenH, screenW } from '$lib/src/stores';
	import Menu from '$lib/components/Menu.svelte';

	let menuOpen = sessionStorage.getItem('menuOpen') === 'true';
	$: sessionStorage.setItem('menuOpen', menuOpen.toString());

	if (!Cookies.get('color_scheme')) {
		Cookies.set('color_scheme', DEFAULT_THEME);
	}

	theme.set(Cookies.get('color_scheme') || DEFAULT_THEME);
	theme.subscribe((value: string) => {
		Cookies.set('color_scheme', value);

		switch (value) {
			case 'light':
			case 'dark':
				colorScheme.set(value);
				break;
			case 'auto':
				colorScheme.set(
					window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
				);
				break;
			default:
				theme.set(DEFAULT_THEME);
				break;
		}
	});
</script>

<svelte:window
	bind:innerHeight={$screenH}
	bind:innerWidth={$screenW}
/>

<SvelteUIProvider withGlobalStyles themeObserver={$colorScheme}>
	<div class="outter">
		<div class="shell">
			<div class="sticky padding">

				<Header height={80} slot="header">
					<HeadContent bind:menuOpen />
				</Header>
			</div>
			<div class="app" id="app">
				<Menu bind:menuOpen />
				<div class="inner-app margin" id="inner-app">
					<slot />
				</div>
			</div>
		</div>
</div>
</SvelteUIProvider>

<style lang="scss">

	.sticky {
		position: sticky;
		top: 0;
		z-index: 100;
	}

	.padding {
		padding: 16px;
		padding-bottom: 0;
	}

	.margin {
		margin: 16px;
	}

	.app {
		flex: 1 1 auto;
		height: 0px;
		margin: 8px;
		margin-top: 0;
		display: flex;
		flex-direction: column;
		position: relative;
	}

	.inner-app {
		position: relative;
		overflow: scroll;
	}

	.shell {
		display: flex;
		flex-direction: column;
		overflow: hidden;
		height: 100dvh;
	}
</style>
