<script lang="ts">
	import { mobile } from '$lib/src/stores';
	import { Burger, Text } from '@svelteuidev/core';
	import IconLogo from '~icons/custom/logo'

	export let menuOpen: boolean;
</script>

<div class="flex-container">
	<div class="flex-item">
		<Burger opened={menuOpen} on:click={() => (menuOpen = !menuOpen)} />
	</div>
	<div class="flex-item">
		{#if !$mobile}
			<Text size={35}>UMass All Campus Makerspace</Text>
		{/if}
	</div>
	<div class="flex-item">
		{#if $mobile}
			<div class="icon">
				<IconLogo height={50} width={50} />
			</div>
		{/if}
	</div>
</div>

<style lang="scss">
	.flex-container {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0 10px;
		width: 100%;
		height: 100%;
	}

	.flex-item {
		flex: 1;
		display: flex;
		justify-content: center;
	}

	.flex-item:first-child {
		flex: 0;
		justify-content: flex-start;
	}

	.flex-item:last-child {
		flex: 0;
		justify-content: flex-end;
	}

	.icon {
		padding-right: 1em;
	}
</style>
