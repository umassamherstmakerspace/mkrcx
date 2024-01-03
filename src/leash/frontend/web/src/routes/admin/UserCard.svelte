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
		Text,
		type SvelteUIColor,
		Menu,
		TypographyProvider
	} from '@svelteuidev/core';
	import {} from '@svelteuidev/motion';
	import { slide } from 'svelte/transition';
	import { quadIn, quadOut } from 'svelte/easing';
	import UserInfo from './UserInfo.svelte';
	import { initalizeUserInfo } from './userCard';
	import {
		Camera,
		ChatBubble,
		CrossCircled,
		Gear,
		MagnifyingGlass,
		Trash,
		Width
	} from 'radix-icons-svelte';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import type { User } from '$lib/src/leash';

	export let user: User;
	let active = false;

	let refresh = {};

	let enabled: SvelteUIColor = 'orange';

	function toggle() {
		active = !active;

		if (active) {
			dispatchEvent(new CustomEvent('userCardOpen', { detail: user }));
		}
	}
</script>

<Paper shadow="sm" padding="lg">
	<Stack>
		<!-- <div class="float">
			<button class="fill" />
		</div> -->
		<UnstyledButton on:click={toggle}>
			<TypographyProvider>
				<Grid cols={24}>
					<Grid.Col span={2}>
						<Badge color={'green'} radius="md" variant="filled">
							Active
						</Badge>
					</Grid.Col>
					<Grid.Col span={4}><Timestamp time={user.createdAt} /></Grid.Col>
					<Grid.Col span={6}>{user.name}</Grid.Col>
					<Grid.Col span={6}>{user.email}</Grid.Col>
					<Grid.Col span={2}>{user.role}</Grid.Col>
					<Grid.Col span={2}>{user.type}</Grid.Col>
					<Grid.Col span={1}>{user.id}</Grid.Col>
					<Grid.Col span={1}>
						<Menu>
							<Menu.Label>Application</Menu.Label>
							<Menu.Item icon={Gear} on:click={() => alert('test')}>Settings</Menu.Item>
							<Menu.Item icon={ChatBubble}>Messages</Menu.Item>
							<Menu.Item icon={Camera}>Gallery</Menu.Item>
							<Menu.Item icon={MagnifyingGlass}>
								<svelte:fragment slot="rightSection">
									<Text size="xs" color="dimmed">⌘K</Text>
								</svelte:fragment>
								Search
							</Menu.Item>

							<Divider />

							<Menu.Label>Danger zone</Menu.Label>
							<Menu.Item icon={Width}>Transfer my data</Menu.Item>
							<Menu.Item color="red" icon={Trash}>Delete my account</Menu.Item>
						</Menu>
					</Grid.Col>
				</Grid>
			</TypographyProvider>
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

<style lang="scss">
	.float {
	}

	.fill {
		width: 100%;
		height: 100%;
	}
</style>
