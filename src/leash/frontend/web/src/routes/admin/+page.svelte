<script lang="ts">
	import { browser } from '$app/environment';

	import { CrossCircled } from 'radix-icons-svelte';
	import UserCard from './UserCard.svelte';
	import {
		Alert,
		Center,
		Grid,
		Input,
		Loader,
		ObserverRender,
		Paper,
		Seo,
		Skeleton,
		Stack
	} from '@svelteuidev/core';

	import type { Snapshot } from './$types';
	import { User } from '$lib/src/leash';

	let query = '';

	let loadKey = {};
	let userSet = new Set();
	let users: User[] = [];
	let needMore = false;

	export const snapshot: Snapshot = {
		capture: () => {
			return {
				query
			};
		},
		restore: (value) => {
			query = value.query;
		}
	};

	async function search(query: string) {
		if (browser) {
			userSet = new Set();

			let res = await User.search(query);
			users = res.data;
			loadKey = {};

			needMore = res.data.length == 100;
		}
	}

	async function loadMoreUsers(): Promise<void> {
		const newUsers = await User.search(query, {
			limit: 100,
			offset: users.length
		});
		users = [
			...users,
			...newUsers.data.filter((user) => {
				if (userSet.has(user.id)) {
					return false;
				} else {
					userSet.add(user.id);
					return true;
				}
			})
		];

		needMore = newUsers.data.length == 100;
	}
</script>

<Seo title="User Directory" description="Search for users in the system." />
<div class="sticky">
	<div class="flex">
		<div class="full-grow">
			<Input bind:value={query} placeholder="Search for users..." />
		</div>
	</div>
	<Paper shadow="sm" padding="lg">
		<Grid>
			<Grid.Col span={1}>User Status</Grid.Col>
			<Grid.Col span={2}>Creation Date</Grid.Col>
			<Grid.Col span={3}>Name</Grid.Col>
			<Grid.Col span={3}>Email</Grid.Col>
			<Grid.Col span={1}>User Type</Grid.Col>
			<Grid.Col span={1}>User Role</Grid.Col>
			<Grid.Col span={1}>User ID</Grid.Col>
		</Grid>
	</Paper>
</div>
<Stack>
	{#await search(query)}
		<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
		<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
		<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
	{:then}
		{#each users as user}
			<UserCard {user} />
		{/each}

		{#if needMore}
			{#key loadKey}
				<ObserverRender let:visible>
					{#if visible}
						{#await loadMoreUsers()}
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
							<Skeleton height={8} radius="xl" override={{ marginTop: '8px' }} />
						{:then}
							<Center>
								<Loader />
							</Center>
						{:catch error}
							<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
								{error.message}
							</Alert>
						{/await}
					{/if}
				</ObserverRender>
			{/key}
		{/if}
	{:catch error}
		<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
			{error.message}
		</Alert>
	{/await}
</Stack>

<style lang="scss">
	.flex {
		display: flex;
		gap: 1rem;
	}

	.full-grow {
		flex-grow: 1;
	}

	.sticky {
		position: sticky;
		top: 0;
		z-index: 100;
	}
</style>
