<script lang="ts">
	import { page } from '$app/stores';
	import { getUserById } from '$lib/src/leash';
	import { timestampCreator } from '$lib/src/stores';
	import type { User } from '$lib/src/types';
	import { Alert, Seo, Switch, Skeleton, TextInput, Stack, InputWrapper } from '@svelteuidev/core';
	import { CrossCircled } from 'radix-icons-svelte';
	import type { Readable } from 'svelte/store';

	let id = Number.parseInt($page.url.searchParams.get('id') ?? '');

	let user: User;

	let firstName: string;
	let lastName: string;
	let email: string;
	let enabled: boolean;
	

	let createdAt: Readable<string>;
	let updatedAt: Readable<string>;

	async function getUser(id: number) {
		user = await getUserById(id);

		firstName = user.firstName;
		lastName = user.lastName;
		email = user.email;
		enabled = user.enabled;

		createdAt = timestampCreator(user.createdAt);
		updatedAt = timestampCreator(user.updatedAt);
	}

	
</script>

<Seo title="User Directory" description="Search for users in the system." />

{#if Number.isInteger(id)}
	{#await getUser(id)}
		<Skeleton />
	{:then}
		<Stack>
		<TextInput label="First Name or Service Name" bind:value={firstName}/>
		<TextInput label="Last Name" bind:value={lastName}/>
		<TextInput label="Email" bind:value={email}/>
		<InputWrapper label="Account Enabled">
			<Switch label={enabled ? "Enabled" : "Disabled"} bind:checked={enabled} />
		</InputWrapper>

		<TextInput label="ID" disabled value={user.id} />
		<TextInput label="Join Date" disabled value={$createdAt} />
		<TextInput label="Last Updated" disabled value={$updatedAt} />
	</Stack>
		<!-- <SimpleGrid cols={2}>
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
		</SimpleGrid> -->
	{:catch error}
		<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
			{error.message}
		</Alert>
	{/await}
{:else}
	<Alert icon={CrossCircled} title="Error" color="red" variant="filled">Invalid user id.</Alert>
{/if}
