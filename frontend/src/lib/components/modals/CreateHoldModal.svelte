<script lang="ts">
    import { User } from '$lib/leash';
    import { getUnixTime } from 'date-fns';
	import { Alert, Button, Input, Label, Modal, NumberInput } from "flowbite-svelte";

    export let user: User;
    export let onConfirm: () => Promise<void> = async () => {};
    export let open: boolean;

	function closeModal() {
		open = false;
	}

    async function confirm() {
		const holdStart = startDate
			? getUnixTime(startDate)
			: undefined;
		const holdEnd = endDate ? getUnixTime(endDate) : undefined;

		try {
			if (holdStart && holdEnd && holdStart > holdEnd) {
				throw new Error('Start date must be before end date');
			}

			await user.createHold({
				holdType,
				reason,
				priority,
				holdStart,
				holdEnd
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

	let holdType = '';
	let reason = '';
	let priority = 0;
	let startDate: Date | undefined = undefined;
	let endDate: Date | undefined = undefined;

	let error = '';

	function reset() {
		holdType = '';
		reason = '';
		priority = 0;
		startDate = undefined;
		endDate = undefined;

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
	<form
		class="flex flex-col space-y-6"
		method="dialog"
		on:submit|preventDefault={confirm}
	>
		<h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">
			Create hold for {user.name}
		</h3>
		<Label class="space-y-2">
			<span>Hold Type</span>
			<Input
				type="text"
				name="text"
				placeholder="Hold Type"
				required
				bind:value={holdType}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Reason</span>
			<Input
				type="text"
				name="text"
				placeholder="Reason"
				required
				bind:value={reason}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Priority</span>
			<NumberInput
				type="number"
				name="text"
				placeholder="Priority"
				required
				bind:value={priority}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Start Date</span>
			<Input
				type="datetime-local"
				name="text"
				placeholder="Start Date"
				bind:value={startDate}
			/>
		</Label>
		<Label class="space-y-2">
			<span>End Date</span>
			<Input
				type="datetime-local"
				name="text"
				placeholder="End Date"
				bind:value={endDate}
			/>
		</Label>
		<Button class="w-full1" type="submit">Create Hold</Button>
	</form>
</Modal>