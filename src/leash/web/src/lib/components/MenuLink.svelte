<script lang="ts">
	import type { SvelteComponent } from 'svelte';
	import { createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher();

	export let href: string;
	export let title: string;
	export let icon: typeof SvelteComponent;
	let active = location.pathname === href;
</script>

<li class={active ? 'active' : ''}>
	<a data-sveltekit-reload {href} on:click={() => dispatch('click')}>
		<svelte:component this={icon} />
		{title}
	</a>
</li>

<style lang="scss">
	li {
		list-style: none;
		border-radius: 0.25rem;
	}

	:global(.dark-theme) li {
		border-left-color: var(--svelteui-colors-dark600);
	}

	li.active {
		background-color: var(--svelteui-colors-blue50);
	}

	:global(.dark-theme) li.active {
		background-color: #1864ab73;
	}

	a {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--svelteui-colors-black);
		text-decoration: none;
		padding: 0.75rem 1rem;
	}

	.active a {
		color: var(--svelteui-colors-primary);
	}

	:global(.dark-theme) a {
		color: var(--svelteui-colors-dark200);
	}

	.active a,
	a:hover {
		color: var(--svelteui-colors-primary);
	}

	:global(.dark-theme) .active a {
		color: var(--svelteui-colors-white);
	}
</style>
