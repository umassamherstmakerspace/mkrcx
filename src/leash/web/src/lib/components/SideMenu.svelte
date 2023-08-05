<script lang="ts">
	import { NativeSelect } from '@svelteuidev/core';
	import { clickoutside, lockscroll } from '@svelteuidev/composables';
	import { theme, user } from '$lib/src/stores';
	import { Role } from '$lib/src/types';
	import MenuLink from './MenuLink.svelte';
	import { Exit, Home, ListBullet, Person } from 'radix-icons-svelte';

	export let menu: boolean;
	export let transitioning: boolean;
</script>

<div
	class="box"
	use:clickoutside={{ enabled: menu && !transitioning, callback: () => (menu = false) }}
	use:lockscroll={menu}
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
