<script lang="ts">
	import { User, permissionOptions } from '$lib/leash';
	import { Alert, Button, Checkbox, Input, Label, Modal, MultiSelect } from 'flowbite-svelte';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		try {
			await user.createAPIKey({
				description,
				fullAccess,
				permissions,
			});

			closeModal();

			await onConfirm();
		} catch (e) {
			if (e instanceof Error) {
				error = e.message;
			} else {
				error = new String(e).toString();
			}
			return;
		}
	}

	let description = '';
	let fullAccess = false;
	let permissions: string[] = [];

	let error = '';

	function reset() {
		description = '';
		fullAccess = false;
		permissions = [];

		error = '';
	}

	$: if (open) reset();
</script>

<Modal bind:open size="xs" autoclose={false} class="w-full">
	{#if error}
		<Alert border color="red" dismissable on:close={() => (error = '')}>
			<span class="font-medium">Error: </span>
			{error}
		</Alert>
	{/if}
	<form class="flex flex-col space-y-6" method="dialog" on:submit|preventDefault={confirm}>
		<h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">
			Create api key for {user.name}
		</h3>
		<div class="flex flex-col justify-between">
			<Label
				for="description-input"
				class="mb-2 block">Description
			</Label>

			<Input
				bind:value={description}
				type="text"
				id="description-input"
			/>
		</div>
		<div class="flex flex-col justify-between">
			<Label
				for="full-access-checkbox"
				class="mb-2 block">Full Access
			</Label>

			<Checkbox
				bind:checked={fullAccess}
				id="full-access-checkbox"
			/>
		</div>
		<div class="flex flex-col justify-between">
			<Label
				for="permissions-select"
				class="mb-2 block">Permissions
			</Label>

			<MultiSelect
				bind:value={permissions}
				items={permissionOptions}
				id="permissions-select"
			></MultiSelect>
		</div>
		<Button class="w-full1" type="submit">Create API Key</Button>
	</form>
</Modal>
