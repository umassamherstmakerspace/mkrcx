<script lang="ts">
	import { NativeSelect } from '@svelteuidev/core';
	import { clickoutside, lockscroll } from '@svelteuidev/composables';
	import { theme, user } from '$lib/src/stores';
	import { Role } from '$lib/src/types';
	import MenuLink from './MenuLink.svelte';
	import { Exit, Home, ListBullet, Person } from 'radix-icons-svelte';

	export let menuOpen: boolean;
	export let transitioning: boolean;
</script>



<style lang="scss">
	.box {
		width: 100%;
        min-width: 400px;
		height: 100%;
		background-color: #ffffff;
		white-space: nowrap;
        display: flex;
        flex-direction: column;
	}

    :global(.dark-theme) .box {
        background-color: var(--svelteui-colors-dark700);
    }

	.inner {
        flex: 1 1 auto;
		padding: 20px;
        display: flex;
        height: 0px;
        flex-direction: column;
	}

    .top {
        flex-grow: 0;
        display: flex;
        flex-direction: column;
    }

    .bottom {
        flex-grow: 1;
        display: flex;
        flex-direction: column;
        justify-content: flex-end;
    }
</style>
