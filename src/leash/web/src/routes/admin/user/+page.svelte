<script lang="ts">
	import { page } from '$app/stores';
	import { getSelf, getUserById } from '$lib/src/leash';
	import { timestampCreator, user as s } from '$lib/src/stores';
	import { Role, type User } from '$lib/src/types';
	import {
		Alert,
		Seo,
		Switch,
		Skeleton,
		TextInput,
		Stack,
		InputWrapper,
		NativeSelect,
		Divider,
		NumberInput
	} from '@svelteuidev/core';
	import { CrossCircled } from 'radix-icons-svelte';
	import type { Readable } from 'svelte/store';

	let id = Number.parseInt($page.url.searchParams.get('id') ?? '');

	let self: User;
	let canEdit: boolean;

	let user: User;

	let name: string;
	let email: string;
	let enabled: boolean;
	let accountType: string;
	let accountRole: string;

	let createdAt: Readable<string>;
	let updatedAt: Readable<string>;

	async function getUser(id: number) {
		self = await getSelf();
		canEdit = self.roleNumber >= Role.USER_ROLE_ADMIN;

		user = await getUserById(id);

		name = user.name;
		email = user.email;
		enabled = user.enabled;
		accountType = user.type;
		accountRole = user.role;

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
			<TextInput label="Full Name" bind:value={name} />
			<TextInput label="Email" bind:value={email} />
			<InputWrapper label="Account Enabled">
				<Switch label={enabled ? 'Enabled' : 'Disabled'} bind:checked={enabled} />
			</InputWrapper>
			<NativeSelect
				data={[
					{ value: 'member', label: 'Member' },
					{ value: 'volunteer', label: 'Makerspace Volunteer' },
					{ value: 'staff', label: 'Makerspace Staff' },
					{ value: 'admin', label: 'Admin' },
					{ value: 'service', label: 'Service Account' }
				]}
				disabled={!canEdit}
				bind:value={accountRole}
				label="Account Role"
			/>
			<NativeSelect
				data={[
					{ value: 'undergrad', label: 'Undergrad Student' },
					{ value: 'grad', label: 'Graduate Student' },
					{ value: 'faculty', label: 'Faculty' },
					{ value: 'staff', label: 'Makerspace Staff' },
					{ value: 'alumni', label: 'Alumni' },
					{ value: 'other', label: 'Other' }
				]}
				bind:value={accountType}
				label="Account Type"
			/>
			<TextInput label="Major" value={user.major} />
			<NumberInput label="Graduation Year" value={user.graduationYear} />
			<Divider />
			<TextInput label="ID" disabled value={user.id} />
			<TextInput label="Join Date" disabled value={$createdAt} />
			<TextInput label="Last Updated" disabled value={$updatedAt} />
		</Stack>
	{:catch error}
		<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
			{error.message}
		</Alert>
	{/await}
{:else}
	<Alert icon={CrossCircled} title="Error" color="red" variant="filled">Invalid user id.</Alert>
{/if}
