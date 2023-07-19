<script lang="ts">
	import {
		Alert,
		Badge,
		Divider,
		Grid,
		Paper,
		Skeleton,
		Stack,
		UnstyledButton,
		type SvelteUIColor
	} from '@svelteuidev/core';
	import {} from '@svelteuidev/motion';
	import type { User } from '$lib/src/types';
	import { slide } from 'svelte/transition';
	import { quadIn, quadOut } from 'svelte/easing';
	import UserInfo from './UserInfo.svelte';
	import { initalizeUserInfo } from './userCard';
	import { CrossCircled } from 'radix-icons-svelte';
	import Timestamp from '$lib/components/Timestamp.svelte';

	export let user: User;
	let active = false;

	let refresh = {};

	let enabled: SvelteUIColor = 'orange';

	if (user.deletedAt) {
		enabled = 'gray';
	} else if (user.enabled) {
		enabled = 'green';
	} else {
		enabled = 'red';
	}

	function toggle() {
		active = !active;

		if (active) {
			dispatchEvent(new CustomEvent('userCardOpen', { detail: user }));
		}
	}
</script>

<Paper shadow="sm" padding="lg" on:click={toggle}>
	<Stack>
		<UnstyledButton on:click={toggle}>
			<Grid>
				<Grid.Col md={1}>
					<Badge color={enabled} radius="md" variant="filled">
						{user.deletedAt ? 'Deleted' : user.enabled ? 'Enabled' : 'Disabled'}
					</Badge>
				</Grid.Col>
				<Grid.Col span={2}><Timestamp time={user.createdAt} /></Grid.Col>
				<Grid.Col span={3}>{user.name}</Grid.Col>
				<Grid.Col span={3}>{user.email}</Grid.Col>
				<Grid.Col span={1}>{user.type}</Grid.Col>
				<Grid.Col span={1}>{user.role}</Grid.Col>
				<Grid.Col span={1}>{user.id}</Grid.Col>
			</Grid>
		</UnstyledButton>

		{#if active}
			<div
				in:slide={{ duration: 300, easing: quadIn }}
				out:slide={{ duration: 200, easing: quadOut }}
			>
				<Divider />
				{#key refresh}
					{#await initalizeUserInfo(user)}
						<div
							in:slide={{ duration: 300, easing: quadIn }}
							out:slide={{ duration: 200, easing: quadOut }}
						>
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
						</div>
					{:then userInfo}
						<div
							in:slide={{ duration: 300, easing: quadIn }}
							out:slide={{ duration: 200, easing: quadOut }}
						>
							<UserInfo {userInfo} on:refresh={() => (refresh = {})} />
						</div>
					{:catch error}
						<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
							{error.message}
						</Alert>
					{/await}
				{/key}
			</div>
		{/if}
	</Stack>
</Paper>
