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
			await user.createNotification({
				title,
				message,
				link,
				group
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

	let title = '';
	let message = '';
	let link = '';
	let group = '';

	let error = '';

	function reset() {
		title = '';
		message = '';
		link = '';
		group = '';

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
			Create notification for {user.name}
		</h3>
		<div class="flex flex-col justify-between">
			<Label for="title-input" class="mb-2 block">Title</Label>

			<Input bind:value={title} id="title-input" type="text" />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="message-input" class="mb-2 block">Message</Label>

			<Input bind:value={message} id="message-input" type="text" />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="link-input" class="mb-2 block">Link</Label>

			<Input bind:value={link} id="link-input" type="text" />
		</div>
		<div class="flex flex-col justify-between">
			<Label for="group-input" class="mb-2 block">Group</Label>

			<Input bind:value={group} id="group-input" type="text" />
		</div>
		<Button class="w-full1" type="submit">Create Notification</Button>
	</form>
</Modal>
