<script lang="ts">
	import HeadContent from '$lib/components/HeadContent.svelte';
	import { AppShell, Header, SvelteUIProvider, Title } from '@svelteuidev/core';
	import { quadIn, quadOut } from 'svelte/easing';
	import { slide } from 'svelte/transition';
	import { lockscroll } from '@svelteuidev/composables';
	import { colorScheme } from '@svelteuidev/core';
	import Cookies from 'js-cookie';
	import { DEFAULT_THEME } from '$lib/src/defaults';
	import { theme } from '$lib/src/stores';

	let menu = false;
	const toggleMenu = () => {
		menu = !menu;
	};

	$: if (menu) {
		openMenu();
	} else {
		closeMenu();
	}

	const lock = lockscroll(document.body);

	const openMenu = () => {
		lock.update ? true : '';
		menu = true;
	};

	const closeMenu = () => {
		lock.update ? false : '';
		menu = false;
	};

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
	<AppShell>
		<div class="sticky">
			<Header height={80} slot="header">
				<HeadContent bind:menu />
			</Header>
		</div>
		{#if menu}
			<div
				class="fullscreen dimmed"
				on:click={toggleMenu}
				on:keydown={toggleMenu}
				role="dialog"
				aria-modal="true"
				aria-hidden="true"
			>
				<div
					class="menu"
					in:slide={{ duration: 300, easing: quadIn, axis: 'x' }}
					out:slide={{ duration: 200, easing: quadOut, axis: 'x' }}
				>
					<div class="innerMenu">
						<Title>Menu</Title>
					</div>
				</div>
			</div>
		{/if}
		<slot />
	</AppShell>
</SvelteUIProvider>

<style lang="scss">
	.fullscreen {
		position: fixed;
		top: 80px;
		left: 0;
		bottom: 0;
		right: 0;
		height: 100vh;
		width: 100vw;
		z-index: 1000;
		overflow: hidden;
	}

	.dimmed {
		background-color: rgba(0, 0, 0, 0.5);
	}

	.menu {
		height: 100%;
		position: fixed;
		top: 80px;
		display: inline-block;
		background-color: white;
	}

	.sticky {
		position: sticky;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 100;
	}

	.innerMenu {
		height: 100%;
		width: 100%;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		padding: 20px;
	}
</style>
