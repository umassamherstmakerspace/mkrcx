<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import CreateHoldModal from '$lib/components/modals/CreateHoldModal.svelte';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import type { ModalOptions } from '$lib/components/modals/modals';
	import type { Hold, User } from '$lib/leash';
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

	let holds = {};
	export let showDeleted = false;

	async function getHolds(showDeleted: boolean): Promise<Hold[]> {
		const holds = await target.getAllHolds(showDeleted, true);
		return holds.sort((a, b) => {
			const aLevel = a.activeLevel();
			const bLevel = b.activeLevel();

			if (aLevel < bLevel) {
				return -1;
			} else if (aLevel > bLevel) {
				return 1;
			} else {
				if (a.priority < b.priority) {
					return -1;
				} else if (a.priority > b.priority) {
					return 1;
				} else {
					return 0;
				}
			}
		});
	}

	let createHoldModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteHoldModal: DeleteModalOptions = {
		open: false,
		name: '',
		deleteFn: async () => {},
		onConfirm: async () => {}
	};

	function createHold() {
		createHoldModal = {
			open: true,
			onConfirm: async () => {
				holds = {};
			}
		};
	}

	function deleteHold(hold: Hold) {
		deleteHoldModal = {
			open: true,
			name: hold.name,
			deleteFn: () => hold.delete(),
			onConfirm: async () => {
				holds = {};
			}
		};
	}
</script>

<CreateHoldModal
	bind:open={createHoldModal.open}
	user={target}
	onConfirm={createHoldModal.onConfirm}
/>

<DeleteModal
	bind:open={deleteHoldModal.open}
	modalType="Hold"
	name={deleteHoldModal.name}
	user={target}
	deleteFn={deleteHoldModal.deleteFn}
	onConfirm={deleteHoldModal.onConfirm}
/>

<div class="flex flex-col space-y-4 pb-4 md:flex-row md:items-center md:justify-between md:gap-4">
	<Button color="primary" class="mb-4 flex-grow md:mb-0 md:w-1/3" on:click={createHold}>
		Create Hold
	</Button>
	<Label class="mt-4 flex flex-grow items-center font-bold md:w-2/3 md:justify-end">
		<Checkbox bind:checked={showDeleted} />
		<span class="mr-2">Show Deleted</span>
	</Label>
</div>

<Table>
	<TableHead>
		<TableHeadCell>Active</TableHeadCell>
		<TableHeadCell>Hold Type</TableHeadCell>
		<TableHeadCell>Reason</TableHeadCell>
		<TableHeadCell>Start Date</TableHeadCell>
		<TableHeadCell>End Date</TableHeadCell>
		<TableHeadCell>Resolution Link</TableHeadCell>
		<TableHeadCell>Date Added</TableHeadCell>
		<TableHeadCell>Added By</TableHeadCell>
		<TableHeadCell>Date Removed</TableHeadCell>
		<TableHeadCell>Removed By</TableHeadCell>
		<TableHeadCell>Remove</TableHeadCell>
	</TableHead>
	<TableBody>
		{#key holds}
			{#await getHolds(showDeleted)}
				<TableBodyRow>
					<TableBodyCell colspan="11" class="p-0">Loading...</TableBodyCell>
				</TableBodyRow>
			{:then holds}
				{#each holds as hold}
					<TableBodyRow>
						<TableBodyCell>
							{#if hold.isActive()}
								<Badge color="green" rounded class="px-2.5 py-0.5">
									<Indicator color="green" size="xs" class="me-1" />Active
								</Badge>
							{:else if hold.isPending()}
								<Badge color="yellow" rounded class="px-2.5 py-0.5">
									<Indicator color="yellow" size="xs" class="me-1" />Pending
								</Badge>
							{:else}
								<Badge color="red" rounded class="px-2.5 py-0.5">
									<Indicator color="red" size="xs" class="me-1" />Deleted
								</Badge>
							{/if}
						</TableBodyCell>
						<TableBodyCell>{hold.name}</TableBodyCell>
						<TableBodyCell>{hold.reason}</TableBodyCell>
						<TableBodyCell>
							{#if hold.start}
								<Timestamp timestamp={hold.start} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if hold.end}
								<Timestamp timestamp={hold.end} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if hold.resolutionLink}
								<a href={hold.resolutionLink} target="_blank" rel="noopener noreferrer">
									{hold.resolutionLink}
								</a>
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell><Timestamp timestamp={hold.createdAt} /></TableBodyCell>
						<TableBodyCell>
							<UserCell user={hold.getAddedBy()} />
						</TableBodyCell>
						<TableBodyCell>
							{#if hold.deletedAt}
								<Timestamp timestamp={hold.deletedAt} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if hold.deletedAt}
								<UserCell user={hold.getRemovedBy()} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if hold.isActive()}
								<CloseButton on:click={() => deleteHold(hold)} />
							{:else}
								-
							{/if}
						</TableBodyCell>
					</TableBodyRow>
				{/each}
			{:catch error}
				<TableBodyRow>
					<TableBodyCell colspan="11" class="p-0">Error: {error.message}</TableBodyCell>
				</TableBodyRow>
			{/await}
		{/key}
	</TableBody>
</Table>
