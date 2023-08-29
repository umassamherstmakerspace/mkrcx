<script lang="ts">
	import { page } from '$app/stores';
	import { getSelf, getUserById, updateUser } from '$lib/src/leash';
	import { timestampCreator, user as s } from '$lib/src/stores';
	import { Role, type User, type LeashUserUpdateRequest } from '$lib/src/types';
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
		NumberInput,
		Button
	} from '@svelteuidev/core';
	import { CrossCircled } from 'radix-icons-svelte';
	import type { Readable } from 'svelte/store';

	let k = {};

	let id = Number.parseInt($page.url.searchParams.get('id') ?? '');

	let self: User;
	let canEdit: boolean;

	let user: User;

	let name: string;
	let email: string;
	let enabled: boolean;
	let accountRole: string;
	let accountType: string;
	let major: string;
	let graduationYear: number;

	let modified = false;

	let createdAt: Readable<string>;
	let updatedAt: Readable<string>;

	async function getUser(id: number) {
		self = await getSelf();
		canEdit = self.roleNumber >= Role.USER_ROLE_ADMIN;

		user = await getUserById(id);

		name = user.name;
		email = user.email;
		enabled = user.enabled;
		accountRole = user.role;
		accountType = user.type;
		major = user.major;
		graduationYear = user.graduationYear;

		createdAt = timestampCreator(user.createdAt);
		updatedAt = timestampCreator(user.updatedAt);
	}

	function modify() {
		modified = true;
	}

	function save() {
		modified = false;
		let req: LeashUserUpdateRequest = {};

		if (name != user.name) {
			req['name'] = name;
		}

		if (email != user.email) {
			req['new_email'] = email;
		}

		if (enabled != user.enabled) {
			req['enabled'] = enabled;
		}

		if (accountRole != user.role) {
			req['role'] = accountRole;
		}

		if (accountType != user.type) {
			req['type'] = accountType;
		}

		if (major != user.major) {
			req['major'] = major;
		}

		if (graduationYear != user.graduationYear) {
			req['grad_year'] = graduationYear;
		}
		
		updateUser(user.email, req);

		k = {};
	}
</script>

<Seo title="User Directory" description="Search for users in the system." />

{#key k}
{#if Number.isInteger(id)}
	{#await getUser(id)}
		<Skeleton />
	{:then}
		<Stack>
			<TextInput label="Full Name" bind:value={name} on:change={modify} on:input={modify}/>
			<TextInput label="Email" bind:value={email} on:change={modify} on:input={modify}/>
			<InputWrapper label="Account Enabled">
				<Switch label={enabled ? 'Enabled' : 'Disabled'} bind:checked={enabled} on:change={modify}/>
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
				on:change={modify}
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
				on:change={modify}
				label="Account Type"
			/>
			<TextInput label="Major" bind:value={major} on:change={modify} on:input={modify}/>
			<NumberInput label="Graduation Year" bind:value={graduationYear} on:change={modify}/>
			<Divider />
			<TextInput label="ID" disabled value={user.id} />
			<TextInput label="Join Date" disabled value={$createdAt} />
			<TextInput label="Last Updated" disabled value={$updatedAt} />
			<Button disabled={!modified} color="blue" variant="filled" fullSize on:click={save}>Save</Button>
		</Stack>
	{:catch error}
		<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
			{error.message}
		</Alert>
	{/await}
{:else}
	<Alert icon={CrossCircled} title="Error" color="red" variant="filled">Invalid user id.</Alert>
{/if}
{/key}