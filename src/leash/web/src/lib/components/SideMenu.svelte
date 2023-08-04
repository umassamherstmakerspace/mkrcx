<script lang="ts">
	import { colorScheme, NativeSelect, SvelteUIProvider } from '@svelteuidev/core';
	import { cssvariable, clickoutside, lockscroll } from '@svelteuidev/composables';
	import { theme } from '$lib/src/stores';

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
            <p>This is a test</p>
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
    }

    .bottom {
        flex-grow: 1;
        display: flex;
        flex-direction: column;
        justify-content: flex-end;
    }
</style>
