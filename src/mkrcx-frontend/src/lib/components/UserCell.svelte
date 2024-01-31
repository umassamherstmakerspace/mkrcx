<script lang="ts">
	import { User } from '$lib/leash';
	import { Alert, Avatar, type IndicatorColorType, type IndicatorPlacementType } from 'flowbite-svelte';

	export let user: User | Promise<User>;
	export let dot:
		| undefined
		| {
				color?: IndicatorColorType;
				rounded?: boolean;
				size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
				border?: boolean;
				placement?: IndicatorPlacementType;
				offset?: boolean;
		  } = undefined;
</script>

{#if user instanceof User}
	<div class="flex items-center">
		<Avatar id="avatar-menu" src={user.iconURL} rounded {dot} />
		<div class="ms-3">
			<div class="font-semibold">{user.name}</div>
			<div class="text-gray-500 dark:text-gray-400">{user.email}</div>
		</div>
	</div>
{:else if user instanceof Promise}
{#await user}
<div class="flex items-center">
	<Avatar id="avatar-menu" rounded {dot} />
	<div class="ms-3">
		<div class="font-semibold">Loading...</div>
		<div class="text-gray-500 dark:text-gray-400">Loading...</div>
	</div>
</div>
{:then user}
<div class="flex items-center">
	<Avatar id="avatar-menu" src={user.iconURL} rounded {dot} />
	<div class="ms-3">
		<div class="font-semibold">{user.name}</div>
		<div class="text-gray-500 dark:text-gray-400">{user.email}</div>
	</div>
</div>
{:catch error}
	<Alert color="red" rounded border>
		Error Loading User
	</Alert>
{/await}
{/if}