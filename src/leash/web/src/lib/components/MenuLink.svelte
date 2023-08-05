<script lang="ts">
	import { ActionIcon } from "@svelteuidev/core";
    import type { SvelteComponent } from "svelte";
    import { createEventDispatcher } from 'svelte';
    
    const dispatch = createEventDispatcher();

    export let href: string;
    export let title: string;
    export let icon: typeof SvelteComponent;
    let active = location.pathname === href;
</script>

<li class={active ? "active" : ""}>
    <a data-sveltekit-reload href={href} on:click={() => dispatch('click')}>
        <svelte:component this={icon} />
        {title}
    </a>
</li>

<style lang="scss">
    li {
        list-style: none;
        border-left: 0.2rem solid;
        border-left-color: var(--svelteui-colors-gray300);
        border-radius: 0 0.25rem 0.25rem 0;
    }

    :global(.dark-theme) li {
        border-left-color: var(--svelteui-colors-dark600);
    }

    li.active {
        background-color: var(--svelteui-colors-blue50);
        border-left-color: var(--svelteui-colors-primary);
    }

    :global(.dark-theme) li.active {
        background-color: #1864ab73;
    }

    a {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        color: var(--svelteui-colors-gray700);
        text-decoration: none;
        padding: 0.75rem 1rem;
    }

    .active a {
        color: var(--svelteui-colors-primary);
        font-weight: bold;
    }

    .active a, a:hover {
        color: var(--svelteui-colors-primary);
    }

    :global(.dark-theme) a {
        color: var(--svelteui-colors-dark200);
    }

    :global(.dark-theme) .active a, :global(.dark-theme) a:hover {
        color: var(--svelteui-colors-white);
    }
</style>