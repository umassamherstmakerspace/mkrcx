<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import CreateNotificationModal from '$lib/components/modals/CreateNotificationModal.svelte';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import type { ModalOptions } from '$lib/components/modals/modals';
	import type { Notification, User } from '$lib/leash';
	import {
		Badge,
		Button,
		Checkbox,
		CloseButton,
		Indicator,
		Label,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';

	export let target: User;

	let notifications = {};
	export let showDeleted = false;

	async function getNotifications(showDeleted: boolean): Promise<Notification[]> {
		const notifications = await target.getAllNotifications(showDeleted, true);
		return notifications.sort((a, b) => {
			if (a.deletedAt && b.deletedAt) return b.deletedAt.getTime() - a.deletedAt.getTime();
			if (a.deletedAt) return 1;
			if (b.deletedAt) return -1;
			else return b.createdAt.getTime() - a.createdAt.getTime();
		});
	}

	let createNotificationModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteNotificationModal: DeleteModalOptions = {
		open: false,
		name: '',
		deleteFn: async () => {},
		onConfirm: async () => {}
	};

	function createNotification() {
		createNotificationModal = {
			open: true,
			onConfirm: async () => {
				notifications = {};
			}
		};
	}

	function deleteNotification(notification: Notification) {
		deleteNotificationModal = {
			open: true,
			name: notification.title,
			deleteFn: () => notification.delete(),
			onConfirm: async () => {
				notifications = {};
			}
		};
	}
</script>

<CreateNotificationModal
	bind:open={createNotificationModal.open}
	user={target}
	onConfirm={createNotificationModal.onConfirm}
/>

<DeleteModal
	bind:open={deleteNotificationModal.open}
	modalType="Notification"
	name={deleteNotificationModal.name}
	user={target}
	deleteFn={deleteNotificationModal.deleteFn}
	onConfirm={deleteNotificationModal.onConfirm}
/>

<div class="flex flex-col space-y-4 pb-4 md:flex-row md:items-center md:justify-between md:gap-4">
	<Button color="primary" class="mb-4 flex-grow md:mb-0 md:w-1/3" on:click={createNotification}>
		Create Notification
	</Button>
	<Label class="mt-4 flex flex-grow items-center font-bold md:w-2/3 md:justify-end">
		<Checkbox bind:checked={showDeleted} />
		<span class="mr-2">Show Deleted</span>
	</Label>
</div>

<Table>
	<TableHead>
		<TableHeadCell>Active</TableHeadCell>
		<TableHeadCell>Title</TableHeadCell>
		<TableHeadCell>Message</TableHeadCell>
		<TableHeadCell>Link</TableHeadCell>
		<TableHeadCell>Group</TableHeadCell>
		<TableHeadCell>Date Added</TableHeadCell>
		<TableHeadCell>Added By</TableHeadCell>
		<TableHeadCell>Date Removed</TableHeadCell>
		<TableHeadCell>Remove</TableHeadCell>
	</TableHead>
	<TableBody>
		{#key notifications}
			{#await getNotifications(showDeleted)}
				<TableBodyRow>
					<TableBodyCell colspan="9" class="p-0">Loading...</TableBodyCell>
				</TableBodyRow>
			{:then notifications}
				{#each notifications as notification}
					<TableBodyRow>
						<TableBodyCell>
							{#if notification.deletedAt == undefined}
								<Badge color="green" rounded class="px-2.5 py-0.5">
									<Indicator color="green" size="xs" class="me-1" />Active
								</Badge>
							{:else}
								<Badge color="red" rounded class="px-2.5 py-0.5">
									<Indicator color="red" size="xs" class="me-1" />Deleted
								</Badge>
							{/if}
						</TableBodyCell>
						<TableBodyCell>{notification.title}</TableBodyCell>
						<TableBodyCell>{notification.message}</TableBodyCell>
						<TableBodyCell>{notification.link}</TableBodyCell>
						<TableBodyCell>{notification.group}</TableBodyCell>
						<TableBodyCell><Timestamp timestamp={notification.createdAt} /></TableBodyCell>
						<TableBodyCell>
							<UserCell user={notification.getAddedBy()} />
						</TableBodyCell>
						<TableBodyCell>
							{#if notification.deletedAt}
								<Timestamp timestamp={notification.deletedAt} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if notification.deletedAt == undefined}
								<CloseButton on:click={() => deleteNotification(notification)} />
							{:else}
								-
							{/if}
						</TableBodyCell>
					</TableBodyRow>
				{/each}
			{:catch error}
				<TableBodyRow>
					<TableBodyCell colspan="9" class="p-0">Error: {error.message}</TableBodyCell>
				</TableBodyRow>
			{/await}
		{/key}
	</TableBody>
</Table>
