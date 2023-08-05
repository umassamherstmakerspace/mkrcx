<script lang="ts">
	import { colorScheme, NativeSelect, SvelteUIProvider } from '@svelteuidev/core';
	import { cssvariable, clickoutside, lockscroll } from '@svelteuidev/composables';
	import { theme, user } from '$lib/src/stores';
	import { searchUsers } from '$lib/src/leash';
	import { Role } from '$lib/src/types';

	export let menu: boolean;
	export let transitioned: boolean;

	$: styleVars = {
		colorPrimary:
			$colorScheme === 'dark' ? 'var(--svelteui-colors-dark700)' : 'var(--svelteui-colors-light700)'
	};
</script>

<div
	class="box"
	use:cssvariable={styleVars}
	use:clickoutside={{ enabled: transitioned, callback: () => (menu = false) }}
	use:lockscroll={menu}
>
	<div class="inner">
        <div class="top">
            <a href="/">Home</a>
            <a href="/profile">Profile</a>

            {#if $user && $user.roleNumber >= Role.USER_ROLE_VOLUNTEER}
            <a href="/admin">Admin</a>
            {/if}

            {#if $user}
                <a href="/logout">Logout</a>
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
		background-color: var(--colorPrimary);
		white-space: nowrap;
        display: flex;
        flex-direction: column;
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
