<script lang="ts">
	import { User } from '$lib/leash';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		closeModal();

		await user.createTraining({
			trainingType,
		});

		await onConfirm();
	}

	let trainingType = '';
</script>

<Modal bind:open size="xs" autoclose={false} class="w-full">
	<form class="flex flex-col space-y-6" method="dialog" on:submit|preventDefault={confirm}>
		<h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">
			Create training for {user.name}
		</h3>
		<Label class="space-y-2" for="training-type">
			<span>Training Type</span>
			<Input
				type="text"
				name="training-type"
				placeholder="Training Type"
				required
				bind:value={trainingType}
			/>
		</Label>
		<Button class="w-full1" type="submit">Create Training</Button>
	</form>
</Modal>
