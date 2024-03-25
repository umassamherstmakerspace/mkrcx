<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import CreateTrainingModal from '$lib/components/modals/CreateTrainingModal.svelte';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import type { ModalOptions } from '$lib/components/modals/modals';
	import type { Training, User } from '$lib/leash';
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

	let trainings = {};
	export let showDeleted = false;

	async function getTrainings(showDeleted: boolean): Promise<Training[]> {
		const trainings = await target.getAllTrainings(showDeleted, true);
		return trainings.sort((a, b) => {
			if (a.deletedAt && b.deletedAt) return b.deletedAt.getTime() - a.deletedAt.getTime();
			if (a.deletedAt) return 1;
			if (b.deletedAt) return -1;
			else return b.createdAt.getTime() - a.createdAt.getTime();
		});
	}

	let createTrainingModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteTrainingModal: DeleteModalOptions = {
		open: false,
		name: '',
		deleteFn: async () => {},
		onConfirm: async () => {}
	};

	function createTraining() {
		createTrainingModal = {
			open: true,
			onConfirm: async () => {
				trainings = {};
			}
		};
	}

	function deleteTraining(training: Training) {
		deleteTrainingModal = {
			open: true,
			name: training.name,
			deleteFn: () => training.delete(),
			onConfirm: async () => {
				trainings = {};
			}
		};
	}
</script>

<CreateTrainingModal
	bind:open={createTrainingModal.open}
	user={target}
	onConfirm={createTrainingModal.onConfirm}
/>

<DeleteModal
	bind:open={deleteTrainingModal.open}
	modalType="Training"
	name={deleteTrainingModal.name}
	user={target}
	deleteFn={deleteTrainingModal.deleteFn}
	onConfirm={deleteTrainingModal.onConfirm}
/>

<div class="flex flex-col space-y-4 pb-4 md:flex-row md:items-center md:justify-between md:gap-4">
	<Button color="primary" class="mb-4 flex-grow md:mb-0 md:w-1/3" on:click={createTraining}>
		Create Training
	</Button>
	<Label class="mt-4 flex flex-grow items-center font-bold md:w-2/3 md:justify-end">
		<Checkbox bind:checked={showDeleted} />
		<span class="mr-2">Show Deleted</span>
	</Label>
</div>

<Table>
	<TableHead>
		<TableHeadCell>Active</TableHeadCell>
		<TableHeadCell>Name</TableHeadCell>
		<TableHeadCell>Level</TableHeadCell>
		<TableHeadCell>Date Added</TableHeadCell>
		<TableHeadCell>Added By</TableHeadCell>
		<TableHeadCell>Date Removed</TableHeadCell>
		<TableHeadCell>Removed By</TableHeadCell>
		<TableHeadCell>Remove</TableHeadCell>
	</TableHead>
	<TableBody>
		{#key trainings}
			{#await getTrainings(showDeleted)}
				<TableBodyRow>
					<TableBodyCell colspan="8" class="p-0">Loading...</TableBodyCell>
				</TableBodyRow>
			{:then trainings}
				{#each trainings as training}
					<TableBodyRow>
						<TableBodyCell>
							{#if training.deletedAt == undefined}
								<Badge color="green" rounded class="px-2.5 py-0.5">
									<Indicator color="green" size="xs" class="me-1" />Active
								</Badge>
							{:else}
								<Badge color="red" rounded class="px-2.5 py-0.5">
									<Indicator color="red" size="xs" class="me-1" />Deleted
								</Badge>
							{/if}
						</TableBodyCell>
						<TableBodyCell>{training.name}</TableBodyCell>
						<TableBodyCell>{training.levelString()}</TableBodyCell>
						<TableBodyCell><Timestamp timestamp={training.createdAt} /></TableBodyCell>
						<TableBodyCell>
							<UserCell user={training.getAddedBy()} />
						</TableBodyCell>
						<TableBodyCell>
							{#if training.deletedAt}
								<Timestamp timestamp={training.deletedAt} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if training.deletedAt}
								<UserCell user={training.getRemovedBy()} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if training.deletedAt == undefined}
								<CloseButton on:click={() => deleteTraining(training)} />
							{:else}
								-
							{/if}
						</TableBodyCell>
					</TableBodyRow>
				{/each}
			{:catch error}
				<TableBodyRow>
					<TableBodyCell colspan="8" class="p-0">Error: {error.message}</TableBodyCell>
				</TableBodyRow>
			{/await}
		{/key}
	</TableBody>
</Table>
