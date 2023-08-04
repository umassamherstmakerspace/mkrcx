<script lang="ts">
	import HeadContent from '$lib/components/HeadContent.svelte';
	import SideMenu from '$lib/components/SideMenu.svelte';
	import { Header, SvelteUIProvider } from '@svelteuidev/core';
	import { quadIn, quadOut } from 'svelte/easing';
	import { slide } from 'svelte/transition';
	import { colorScheme } from '@svelteuidev/core';
	import Cookies from 'js-cookie';
	import { DEFAULT_THEME } from '$lib/src/defaults';
	import { theme } from '$lib/src/stores';

	let menu = false;
	let transitioned = false;

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

<SvelteUIProvider withGlobalStyles themeObserver={$colorScheme}>
	<div class="outter">
		<div class="shell">
			<div class="sticky padding">

				<Header height={80} slot="header">
					<HeadContent bind:menu />
				</Header>
			</div>
			<div class="app" id="app">
				{#if menu}
				<div class="fullscreen dimmed">
					<div
						class="menu"
						in:slide={{ duration: 300, easing: quadIn, axis: 'x' }}
						out:slide={{ duration: 200, easing: quadOut, axis: 'x' }}
						on:introend={() => (transitioned = true)}
						on:outrostart={() => (transitioned = false)}
					>
						<SideMenu bind:menu bind:transitioned />
					</div>
				</div>
			{/if}
				<div class="inner-app padding" id="inner-app">
					<slot />
				</div>
			</div>
		</div>
</div>
</SvelteUIProvider>

<style lang="scss">
	.fullscreen {
		position: absolute;
		height: 100%;
		width: 100vw;
		z-index: 1000;
	}

	.dimmed {
		background-color: rgba(0, 0, 0, 0.5);
	}

	.menu {
		height: 100%;
		bottom: 0;
		display: inline-block;
		background-color: white;
	}

	.sticky {
		position: sticky;
		top: 0;
		z-index: 100;
	}

	.padding {
		padding: 16px;
		padding-bottom: 0;
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
		height: 100vh;
	}
</style>
