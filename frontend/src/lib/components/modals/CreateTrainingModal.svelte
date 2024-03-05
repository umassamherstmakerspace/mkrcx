<script lang="ts">
	import { User } from '$lib/leash';
	import { Alert, Button, Input, Label, Modal } from 'flowbite-svelte';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		try {
			if (!trainingType) {
				throw new Error('Training Type is required');
			}

			await user.createTraining({
				trainingType,
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

	let trainingType = '';

	let error = '';

	function reset() {
		trainingType = '';

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
			Create training for {user.name}
		</h3>
		<div class="flex flex-col justify-between">
			<Label
				for="trainingType-input"
				class="mb-2 block">Training Type
			</Label>
			<Input
				bind:value={trainingType}
				type="text"
				placeholder="Training Type"
				id="trainingType-input"
			/>
		</div>
		<Button class="w-full1" type="submit">Create Training</Button>
	</form>
</Modal>
