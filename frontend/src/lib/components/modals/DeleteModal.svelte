<script context="module" lang="ts">
	export type DeleteModalType = 'Training' | 'Hold' | 'Notification' | 'API Key';

	export interface DeleteModalOptions extends ModalOptions {
		name: string;
	}
</script>

<script lang="ts">
	import { User } from '$lib/leash';
	import { Button, Modal } from 'flowbite-svelte';
	import { ExclamationCircleOutline } from 'flowbite-svelte-icons';
	import type { ModalOptions } from './modals';

	export let user: User;
	export let onConfirm: () => Promise<void> = async () => {};
	export let modalType: DeleteModalType;
	export let name: string;
	export let open: boolean;

	function closeModal() {
		open = false;
	}

	async function confirm() {
		closeModal();
		await onConfirm();
	}
</script>

<Modal bind:open size="xs" autoclose>
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
