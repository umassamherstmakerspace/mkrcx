<script lang="ts">
	import { page } from '$app/stores';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import { getUserById } from '$lib/src/leash';
	import type { User } from '$lib/src/types';
	import { Alert, Seo, SimpleGrid, Skeleton, Tabs, Text } from '@svelteuidev/core';
	import { CrossCircled } from 'radix-icons-svelte';

	let id = Number.parseInt($page.url.searchParams.get('id') ?? '');

	let active = 2;

	function onActiveChange(event) {
		const { index, key } = event.detail;
		console.log('Tab active', index, key);
	}

	let user: User;
	let userPlaceholder: User;

	async function getUser(id: number) {
		user = await getUserById(id);
		userPlaceholder = JSON.parse(
			JSON.stringify(user, (key, value) =>
				typeof value === 'function' ? null : typeof value === 'object' ? null : value
			)
		);
	}
</script>

<Seo title="User Directory" description="Search for users in the system." />

{#if Number.isInteger(id)}
	{#await getUser(id)}
		<Skeleton />
	{:then}
		<Tabs bind:active on:change={onActiveChange}>
			<Tabs.Tab label="Gallery">Gallery tab content</Tabs.Tab>
			<Tabs.Tab label="Messages">Messages tab content</Tabs.Tab>
			<Tabs.Tab label="Settings">Settings tab content</Tabs.Tab>
		</Tabs>
		<SimpleGrid cols={2}>
			<Text color="dimmed">Name</Text>
			<Text>{user.name}</Text>

			<Text color="dimmed">ID</Text>
			<Text>{user.id}</Text>

			<Text color="dimmed">Email</Text>
			<Text>{user.email}</Text>

			<Text color="dimmed">Join Date</Text>
			<Text><Timestamp time={user.createdAt} /></Text>

			<Text color="dimmed">Last Updated</Text>
			<Text><Timestamp time={user.updatedAt} /></Text>

			<Text color="dimmed">Enabled</Text>
			<Text>{user.enabled ? 'Yes' : 'No'}</Text>

			<Text color="dimmed">Admin</Text>
			<Text>{user.admin ? 'Yes' : 'No'}</Text>

			<Text color="dimmed">Role</Text>
			<Text>{user.role}</Text>

			<Text color="dimmed">User Type</Text>
			<Text>{user.type}</Text>

			{#if user.graduationYear > 0}
				<Text color="dimmed">Graduation Year</Text>
				<Text>{user.graduationYear}</Text>
			{/if}

			{#if user.major}
				<Text color="dimmed">Major</Text>
				<Text>{user.major}</Text>
			{/if}
		</SimpleGrid>
	{:catch error}
		<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
			{error.message}
		</Alert>
	{/await}
{:else}
	<Alert icon={CrossCircled} title="Error" color="red" variant="filled">Invalid user id.</Alert>
{/if}
