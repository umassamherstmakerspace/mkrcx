<script lang="ts">
	import Branding from './Branding.svelte';
	import { Burger } from '@svelteuidev/core';

	export let menu: boolean;
</script>

<div class="flex-container">
	<div class="flex-item">
		<Burger opened={menu} on:click={() => (menu = !menu)} />
	</div>
	<div class="flex-item">
		<Branding />
	</div>
	<div class="flex-item">
		<!-- <ThemeSwitcher /> -->
	</div>
</div>

<style lang="scss">
	.flex-container {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10px;
		width: 100%;
	}

	.flex-item {
		flex: 1;
		display: flex;
		justify-content: center;
	}

	.flex-item:first-child {
		justify-content: flex-start;
	}

	.flex-item:last-child {
		justify-content: flex-end;
	}
</style>
