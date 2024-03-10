<script lang="ts">
	import { TrainingLevel, User, trainingLevelToString } from '$lib/leash';
	import { Alert, Button, Input, Label, Modal } from 'flowbite-svelte';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		try {
			if (!name) {
				throw new Error('Training Type is required');
			}

			await user.createTraining({
				name,
				level
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

	let name = '';
	let level = TrainingLevel.IN_PROGRESS;

	let error = '';

	function reset() {
		name = '';
		level = TrainingLevel.IN_PROGRESS;

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
			<Label for="name-input" class="mb-2 block">Name</Label>
			<Input bind:value={name} type="text" placeholder="Name" id="name-input" />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="level-input" class="mb-2 block">Level</Label>
			<select
				bind:value={level}
				id="level-input"
				class="w-full rounded-md border border-gray-300 p-2"
			>
				<option value={TrainingLevel.IN_PROGRESS}
					>{trainingLevelToString(TrainingLevel.IN_PROGRESS)}</option
				>
				<option value={TrainingLevel.SUPERVISED}
					>{trainingLevelToString(TrainingLevel.SUPERVISED)}</option
				>
				<option value={TrainingLevel.UNSUPERVISED}
					>{trainingLevelToString(TrainingLevel.UNSUPERVISED)}</option
				>
				<option value={TrainingLevel.CAN_TRAIN}
					>{trainingLevelToString(TrainingLevel.CAN_TRAIN)}</option
				>
			</select>
		</div>
		<Button class="w-full1" type="submit">Create Training</Button>
	</form>
</Modal>
