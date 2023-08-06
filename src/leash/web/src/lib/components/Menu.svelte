<script lang="ts">
	import { slide } from "svelte/transition";
	import { quadIn, quadOut } from "svelte/easing";
	import { mobile, theme, user } from "$lib/src/stores";
	import { clickoutside, lockscroll } from "@svelteuidev/composables";
	import MenuLink from "./MenuLink.svelte";
	import { Exit, Home, ListBullet, Person } from "radix-icons-svelte";
	import { Role } from "$lib/src/types";
	import { NativeSelect } from "@svelteuidev/core";

    export let menuOpen: boolean;
    let menuTransitioning: boolean;

    let axis: "x" | "y" = "x";
    $: axis = $mobile ? "y" : "x";
</script>

{#if menuOpen}
    <div class="fullscreen dimmed">
        <div
            class="menu"
            class:mobile={$mobile}
            in:slide={{ duration: 300, easing: quadIn, axis }}
            out:slide={{ duration: 200, easing: quadOut, axis }}
            on:introstart={() => (menuTransitioning = true)}
            on:introend={() => (menuTransitioning = false)}
            on:outrostart={() => (menuTransitioning = true)}
            on:outroend={() => (menuTransitioning = false)}
        >
        <div
            class="box"
            use:clickoutside={{ enabled: menuOpen && !menuTransitioning, callback: () => (menuOpen = false) }}
            use:lockscroll={menuOpen}
        >
            <div class="inner">
                <div class="top">
                    <MenuLink href="/" title="Home" icon={Home} />
                    <MenuLink href="/profile" title="Profile" icon={Person} />
        
                    {#if $user && $user.roleNumber >= Role.USER_ROLE_VOLUNTEER}
                        <MenuLink href="/admin" title="Admin User Directory" icon={ListBullet} />
                    {/if}
        
                    {#if $user}
                        <MenuLink href="/logout" title="Logout" icon={Exit} />
                    {/if}
                </div>
                <div class="bottom">
                    <NativeSelect
                        data={[
                            { value: 'auto', label: 'Auto' },
                            { value: 'light', label: 'Light' },
                            { value: 'dark', label: 'Dark' }
                        ]}
                        bind:value={$theme}
                        label="Theme"
                    />
                </div>
            </div>
        </div>
        </div>
    </div>
{/if}


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

    .menu.mobile {
        display: block;
    }

    .box {
		width: 100%;
        min-width: 400px;
		height: 100%;
		background-color: #ffffff;
		white-space: nowrap;
        display: flex;
        flex-direction: column;
	}

    .mobile .box {
        min-width: 0;
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