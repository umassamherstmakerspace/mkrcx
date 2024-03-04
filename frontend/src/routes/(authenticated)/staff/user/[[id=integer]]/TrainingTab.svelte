<script lang="ts">
	import Timestamp from "$lib/components/Timestamp.svelte";
	import UserCell from "$lib/components/UserCell.svelte";
	import CreateTrainingModal from "$lib/components/modals/CreateTrainingModal.svelte";
	import DeleteModal, { type DeleteModalOptions } from "$lib/components/modals/DeleteModal.svelte";
	import { timeout, type ModalOptions } from "$lib/components/modals/modals";
	import type { Training, User } from "$lib/leash";
	import { Badge, Button, CloseButton, Indicator, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell } from "flowbite-svelte";

    export let target: User;

    let trainings = {};

    async function getTrainings(): Promise<Training[]> {
		const trainings = await target.getAllTrainings(true, true);
		return trainings.sort((a, b) => {
			if (a.deletedAt && b.deletedAt)
				return b.deletedAt.getTime() - a.deletedAt.getTime();
			if (a.deletedAt)
				return 1;
			if (b.deletedAt)
				return -1;
			else
				return b.createdAt.getTime() - a.createdAt.getTime();
		});
	}

    let createTrainingModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

    let deleteTrainingModal: DeleteModalOptions = {
		open: false,
		name: '',
		onConfirm: async () => {}
	};

    function createTraining() {
		createTrainingModal = {
			open: true,
			onConfirm: async () => {trainings = {}}
		};
	}

    function deleteTraining(training: Training) {
		deleteTrainingModal = {
			open: true,
			name: training.trainingType,
			onConfirm: async () => {
				deleteTrainingModal.open = false;
				await training.delete();
				await timeout(300);
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
	onConfirm={deleteTrainingModal.onConfirm}
/>

<Button color="primary" class="mb-4 w-full" on:click={createTraining}>New Training</Button>
<Table>
	<TableHead>
		<TableHeadCell>Active</TableHeadCell>
		<TableHeadCell>Training Type</TableHeadCell>
		<TableHeadCell>Date Added</TableHeadCell>
		<TableHeadCell>Added By</TableHeadCell>
		<TableHeadCell>Date Removed</TableHeadCell>
		<TableHeadCell>Removed By</TableHeadCell>
		<TableHeadCell>Remove</TableHeadCell>
	</TableHead>
	<TableBody>
		{#key trainings}
			{#await getTrainings()}
				<TableBodyRow>
					<TableBodyCell colspan="7" class="p-0">Loading...</TableBodyCell>
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
						<TableBodyCell>{training.trainingType}</TableBodyCell>
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
					<TableBodyCell colspan="7" class="p-0">Error: {error.message}</TableBodyCell>
				</TableBodyRow>
			{/await}
		{/key}
	</TableBody>
</Table>
