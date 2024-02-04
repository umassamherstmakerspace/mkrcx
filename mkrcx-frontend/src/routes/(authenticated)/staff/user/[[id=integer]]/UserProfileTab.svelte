<script lang="ts">
	import {
		Alert,
		Button,
		Helper,
		Input,
		Label,
		NumberInput,
		Select,
	} from 'flowbite-svelte';
	import type { PageData } from './$types';
	import type { UserUpdateOptions } from '$lib/leash';
	import type { ConvertFields } from '$lib/types';
	import { inputColor, isError, labelColor } from './formCommon';

	export let data: PageData;
	let { user, target } = data;

	let profileError: string = '';
    let changed = false;
	const change = () => (changed = true);

	const userUpdate: UserUpdateOptions = {
		name: '',
		email: '',
		type: '',
		major: undefined,
		graduationYear: undefined,
		role: undefined,
		cardID: undefined
	};

	const userUpdateError: ConvertFields<UserUpdateOptions, string> = {};

	function loadUser() {
		userUpdate.name = target.name;
		userUpdate.email = target.email;
		if (target.pendingEmail) {
			userUpdate.email = target.pendingEmail;
		}
		userUpdate.type = target.type;
		if (target.type == 'undergrad' || target.type == 'grad' || target.type == 'alumni') {
			userUpdate.major = target.major;
			userUpdate.graduationYear = target.graduationYear;
		}

        if (user.role === 'admin') {
            userUpdate.role = target.role;
            userUpdate.cardID = target.cardId;
        } else {
            userUpdate.role = undefined;
            userUpdate.cardID = undefined;
        }

		changed = false;

		Object.keys(userUpdateError).forEach((key) => {
			userUpdateError[key as keyof UserUpdateOptions] = undefined;
		});
	}

	loadUser();

	function validate(): boolean {
		const emailRe =
			/^(([^<>()[\].,;:\s@"]+(\.[^<>()[\].,;:\s@"]+)*)|(".+"))@(([^<>()[\].,;:\s@"]+\.)+[^<>()[\].,;:\s@"]{2,})$/i;
		let hasError = false;
		if (!userUpdate.name) {
			userUpdateError.name = 'Name cannot be empty';
			hasError = true;
		} else {
			userUpdateError.name = undefined;
		}

		if (!userUpdate.email) {
			userUpdateError.email = 'Email cannot be empty';
			hasError = true;
		} else if (!emailRe.test(userUpdate.email)) {
			userUpdateError.email = 'Email is invalid';
			hasError = true;
		} else {
			userUpdateError.email = undefined;
		}

		userUpdateError.major = undefined;

		if (
			!userUpdate.graduationYear ||
			userUpdate.graduationYear < 1900 ||
			userUpdate.graduationYear > new Date().getFullYear() + 10
		) {
			userUpdateError.graduationYear = 'Invalid graduation year';
			hasError = true;
		} else {
			userUpdateError.graduationYear = undefined;
		}

        if (userUpdate.role === 'admin' || userUpdate.role === 'staff' || userUpdate.role === 'volunteer' || userUpdate.role === 'member') {
            userUpdateError.role = undefined;
        } else {
            userUpdateError.role = 'Invalid role';
            hasError = true;
        }

        userUpdateError.cardID = undefined;

		return hasError;
	}

	async function updateUser() {
		if (validate()) return;

		try {
			target = await target.update(userUpdate);
			loadUser();
		} catch (error) {
			console.error(error);
			let message = '';
			if (error instanceof Error) {
				message = error.message;
			} else {
				message = String(error);
			}

			profileError = message;
		}
	}
</script>

{#if profileError}
	<Alert border color="red" dismissable on:close={() => (profileError = '')}>
		<span class="font-medium">Error: </span>
		{profileError}
	</Alert>
{/if}
<form on:submit|preventDefault={updateUser}>
	<div class="flex flex-col space-y-10">
		<div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.name)} for="name-input" class="mb-2 block"
				>Name</Label
			>
			<Input
				color={inputColor(userUpdateError.name)}
				bind:value={userUpdate.name}
				on:input={change}
				on:change={validate}
				type="text"
				id="name-input"
			/>
			{#if isError(userUpdateError.name)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.name}
				</Helper>
			{/if}
		</div>
		<div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.email)} for="email-input" class="mb-2 block"
				>Email</Label
			>
			<Input
				color={inputColor(userUpdateError.email)}
				bind:value={userUpdate.email}
				on:input={change}
				on:change={validate}
				type="email"
				id="email-input"
			/>
			{#if isError(userUpdateError.email)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.email}
				</Helper>
			{/if}
			{#if target.pendingEmail}
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Pending Email: {target.pendingEmail}
				</p>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Current Email: {target.email}
				</p>
			{/if}
		</div>
        {#if user.role === 'admin'}
        <div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.email)} for="cardID-input" class="mb-2 block"
				>Card ID</Label
			>
			<Input
				color={inputColor(userUpdateError.cardID)}
				bind:value={userUpdate.cardID}
				on:input={change}
				on:change={validate}
				type="text"
				id="cardID-input"
			/>
			{#if isError(userUpdateError.cardID)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.cardID}
				</Helper>
			{/if}
		</div>
        {/if}
        <div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.type)} for="type-selector" class="mb-2 block"
				>User Role</Label
			>
            {#if user.role === 'admin'}
                <Select
                    color={inputColor(userUpdateError.role)}
                    bind:value={userUpdate.role}
                    on:input={change}
                    on:change={validate}
                    id="type-selector"
                >
                    <option value="admin">Admin</option>
                    <option value="staff">Staff</option>
                    <option value="volunteer">Volunteer</option>
                    <option value="member">Member</option>
                </Select>
                {#if isError(userUpdateError.role)}
                    <Helper class="mt-2" color="red">
                        <span class="font-medium">Error:</span>
                        {userUpdateError.role}
                    </Helper>
                {/if}
            {:else}
            <Select
				color={inputColor(userUpdateError.role)}
				value={target.role}
				disabled
				id="type-selector"
			  >
                <option value="admin">Admin</option>
                <option value="staff">Staff</option>
                <option value="volunteer">Volunteer</option>
                <option value="member">Member</option>
            </Select>
            {/if}
		</div>
		<div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.type)} for="type-selector" class="mb-2 block"
				>User Type</Label
			>
			<Select
				color={inputColor(userUpdateError.type)}
				bind:value={userUpdate.type}
				on:input={change}
				on:change={validate}
				id="type-selector"
			>
				<option value="undergrad">Undergraduate Student</option>
				<option value="grad">Graduate Student</option>
				<option value="alumni">Alumni</option>
				<option value="faculty">Faculty</option>
				<option value="staff">Staff</option>
				<option value="other">Other</option>
			</Select>
			{#if isError(userUpdateError.type)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.type}
				</Helper>
			{/if}
		</div>
		<div class="flex flex-col justify-between">
			<Label color={labelColor(userUpdateError.major)} for="major-input" class="mb-2 block"
				>Major</Label
			>
			<Input
				color={inputColor(userUpdateError.major)}
				bind:value={userUpdate.major}
				on:input={change}
				on:change={validate}
				type="text"
				id="major-input"
			/>
			{#if isError(userUpdateError.major)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.major}
				</Helper>
			{/if}
		</div>
		<div class="flex flex-col justify-between">
			<Label
				color={labelColor(userUpdateError.graduationYear)}
				for="graduation-year-input"
				class="mb-2 block">Graduation Year</Label
			>
			<NumberInput
				color={inputColor(userUpdateError.graduationYear)}
				bind:value={userUpdate.graduationYear}
				on:input={change}
				on:change={validate}
				type="number"
				id="graduation-year-input"
			/>
			{#if isError(userUpdateError.graduationYear)}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.graduationYear}
				</Helper>
			{/if}
		</div>
		<div class="flex justify-end">
			<Button color="yellow" disabled={!changed} class="w-1/4" type="submit">Save</Button>
		</div>
	</div>
</form>
