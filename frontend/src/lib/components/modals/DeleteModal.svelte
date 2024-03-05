<script context="module" lang="ts">
	export type DeleteModalType = 'Training' | 'Hold' | 'Notification' | 'API Key';

	export interface DeleteModalOptions extends ModalOptions {
		name: string;
		deleteFn: () => Promise<void>;
	}
</script>

<script lang="ts">
	import { User } from '$lib/leash';
	import { Alert, Button, Modal } from 'flowbite-svelte';
	import { ExclamationCircleOutline } from 'flowbite-svelte-icons';
	import { timeout, type ModalOptions } from './modals';

	export let user: User;
	export let deleteFn: () => Promise<void>;
	export let onConfirm: () => Promise<void> = async () => {};
	export let modalType: DeleteModalType;
	export let name: string;
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		await deleteFn();
		closeModal();
		await timeout(300);
		await onConfirm();
	}

	let error = '';

	function reset() {
		error = '';
	}

	$: if (open) reset();
</script>

<Modal bind:open size="xs" autoclose>
	{#if error}
		<Alert border color="red" dismissable on:close={() => (error = '')}>
			<span class="font-medium">Error: </span>
			{error}
		</Alert>
	{/if}
	<div class="text-center">
		<ExclamationCircleOutline class="mx-auto mb-4 h-12 w-12 text-gray-400 dark:text-gray-200" />
		<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
			Are you sure you want to remove the {name}
			{modalType.toLowerCase()} from {user.name}?
		</h3>
		<Button color="red" class="me-2" on:click={confirm}>Remove {modalType}</Button>
		<Button color="alternative" on:click={closeModal}>Cancel</Button>
	</div>
</Modal>
