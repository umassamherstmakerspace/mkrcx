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
		CloseButton,
		Indicator,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';

	export let target: User;

	let notifications = {};

	async function getNotifications(): Promise<Notification[]> {
		const notifications = await target.getAllNotifications(true, true);
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
			deleteFn: notification.delete,
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

<Button color="primary" class="mb-4 w-full" on:click={createNotification}>New Notification</Button>
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
			{#await getNotifications()}
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
