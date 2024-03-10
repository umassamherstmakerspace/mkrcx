<script lang="ts">
	import { User } from '$lib/leash';
	import { getUnixTime } from 'date-fns';
	import { Alert, Button, Input, Label, Modal, NumberInput } from 'flowbite-svelte';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		const start = startDate ? getUnixTime(startDate) : undefined;
		const end = endDate ? getUnixTime(endDate) : undefined;

		try {
			if (!name) {
				throw new Error('Hold type is required');
			}

			if (!reason) {
				throw new Error('Reason is required');
			}

			if (start && end && start > end) {
				throw new Error('Start date must be before end date');
			}

			if (resolutionLink && !resolutionLink.startsWith('http')) {
				throw new Error('Resolution link must be a valid URL');
			}

			await user.createHold({
				name,
				reason,
				priority,
				start,
				end,
				resolutionLink
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
	let reason = '';
	let priority = 0;
	let startDate: Date | undefined = undefined;
	let endDate: Date | undefined = undefined;
	let resolutionLink = '';

	let error = '';

	function reset() {
		name = '';
		reason = '';
		priority = 0;
		startDate = undefined;
		endDate = undefined;
		resolutionLink = '';

		error = '';
	}

	$: if (open) reset();

	$: if (priority < 0) priority = 0;
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
			Create hold for {user.name}
		</h3>
		<div class="flex flex-col justify-between">
			<Label for="name-input" class="mb-2 block">Hold Type</Label>

			<Input type="text" name="text" placeholder="Hold Type" required bind:value={name} />
		</div>

		<div class="flex flex-col justify-between">
			<Label for="reason-input" class="mb-2 block">Reason</Label>

			<Input type="text" name="text" placeholder="Reason" required bind:value={reason} />
		</div>

		<div class="flex flex-col justify-between">
			<Label
				for="priority-input"
				class="mb-2 block
			"
				>Priority
			</Label>

			<NumberInput
				type="number"
				name="text"
				placeholder="Priority"
				required
				bind:value={priority}
			/>
		</div>
		<div class="flex flex-col justify-between">
			<Label for="resolution-link-input" class="mb-2 block">Resolution Link</Label>

			<Input type="text" name="text" placeholder="Resolution Link" bind:value={resolutionLink} />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="start-date-input" class="mb-2 block">Start Date</Label>

			<Input type="datetime-local" name="text" placeholder="Start Date" bind:value={startDate} />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="end-date-input" class="mb-2 block">End Date</Label>

			<Input type="datetime-local" name="text" placeholder="End Date" bind:value={endDate} />
		</div>
		<Button class="w-full1" type="submit">Create Hold</Button>
	</form>
</Modal>
