<script lang="ts">
	import { Alert, Button, Helper, Input, Label, MultiSelect } from 'flowbite-svelte';
	import type { PageData } from './$types';
	import { allPermissions, type ServiceUserUpdateOptions } from '$lib/leash';
	import type { ConvertFields } from '$lib/types';
	import { inputColor, isError, labelColor } from './formCommon';

	export let data: PageData;
	let { user, target } = data;
	const permissionOptions = allPermissions.map((permission) => ({
		name: permission,
		value: permission
	}));

	let profileError: string = '';
	let changed = false;
	const change = () => (changed = true);

	const userUpdate: ServiceUserUpdateOptions = {
		name: '',
		permissions: []
	};

	const userUpdateError: ConvertFields<typeof userUpdate, string> = {};

	function loadUser() {
		userUpdate.name = target.name;
		userUpdate.permissions = target.permissions;

		changed = false;

		Object.keys(userUpdateError).forEach((key) => {
			userUpdateError[key as keyof typeof userUpdate] = undefined;
		});
	}

	loadUser();

	function validate(): boolean {
		let hasError = false;
		if (!userUpdate.name) {
			userUpdateError.name = 'Name cannot be empty';
			hasError = true;
		} else {
			userUpdateError.name = undefined;
		}

		userUpdateError.permissions = undefined;

		return hasError;
	}

	async function updateUser() {
		if (validate()) return;

		try {
			target = await target.updateService(userUpdate);
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
			<Label color={labelColor(userUpdateError.permissions?.toString())} for="permissions-select" class="mb-2 block"
				>Permissions</Label
			>
			<MultiSelect
				color={inputColor(userUpdateError.permissions?.toString())}
				bind:value={userUpdate.permissions}
				items={permissionOptions}
				on:input={change}
				on:change={validate}
				id="permissions-select"
			>
			</MultiSelect>
			{#if isError(userUpdateError.permissions?.toString())}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{userUpdateError.permissions?.toString()}
				</Helper>
			{/if}
		</div>
		<div class="flex justify-end">
			<Button color="yellow" disabled={!changed} class="w-1/4" type="submit">Save</Button>
		</div>
	</div>
</form>
