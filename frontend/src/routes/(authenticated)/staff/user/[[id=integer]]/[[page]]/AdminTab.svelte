<script lang="ts">
	import type { User } from '$lib/leash';
	import {
		Button,
		Helper,
		Input,
		Label,
		Modal,
		P,
	} from 'flowbite-svelte';

	export let target: User;

	let deleteUserModal = {
		open: false,
		confirmation: '',
		error: '',
	}
</script>

<Modal
	bind:open={deleteUserModal.open}
	size="md"
	autoclose={false}
	class="w-full"
>
	<div class="flex flex-col space-y-6">
		<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
			Delete {target.name}?
		</h3>

		<P class="mb-5 text-lg dark:text-gray-400">
			Please type the user's name (<span class="font-medium">{target.name}</span>) to confirm.
		</P>

		<div class="flex flex-col justify-between">
			<Label 
			for="confirmation-input" 
			class="text-sm font-medium text-gray-900 dark:text-white"
			color={deleteUserModal.error ? 'red' : 'gray'}
			>
				Confirmation
			</Label>
			<Input
				type="text"
				name="text"
				placeholder={target.name}
				required
				bind:value={deleteUserModal.confirmation}
				color={deleteUserModal.error ? 'red' : 'base'}
			/>
			{#if deleteUserModal.error}
				<Helper class="mt-2" color="red">
					<span class="font-medium">Error:</span>
					{deleteUserModal.error}
				</Helper>
			{/if}
		</div>

		<Button
			color="red"
			class="me-2"
			on:click={() => {
				if (deleteUserModal.confirmation === target.name) {
					target.delete();
					deleteUserModal.open = false;
				} else {
					deleteUserModal.error = 'Confirmation does not match';
				}
			}}
		>
			Delete {target.name}
		</Button>
	</div>
</Modal>

<div class="flex flex-col items-center justify-center text-center px-16 gap-12">
	<Button
	class="w-full"
	color="red"
	on:click={() => {
		deleteUserModal.open = true;
		deleteUserModal.confirmation = '';
		deleteUserModal.error = '';
	}}
>
	Delete {target.name}
</Button>
</div>