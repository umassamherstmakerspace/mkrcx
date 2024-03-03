<script lang="ts">
	import {
		Tabs,
		TabItem,
		Table,
		TableHead,
		TableHeadCell,
		TableBody,
		TableBodyRow,
		TableBodyCell,
		Badge,
		Indicator,

		Button,

		CloseButton


	} from 'flowbite-svelte';
	import { RectangleListSolid, AnnotationSolid, UserCircleSolid } from 'flowbite-svelte-icons';
	import UserCell from '$lib/components/UserCell.svelte';

	import type { PageData, Snapshot } from './$types';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserProfileTab from './UserProfileTab.svelte';
	import ServiceProfileTab from './ServiceProfileTab.svelte';
	import CreateTrainingModal from '$lib/components/modals/CreateTrainingModal.svelte';
	import { timeout, type ModalOptions } from '$lib/components/modals/modals';
	import type { DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import type { Hold, Training } from '$lib/leash';
	import CreateHoldModal from '$lib/components/modals/CreateHoldModal.svelte';

	export let data: PageData;
	const { api, user, tabs } = data;
	let { tabsOpen, target } = data;

	type Data = {
		tabsOpen: typeof tabsOpen;
	};

	export const snapshot: Snapshot<Data> = {
		capture: () => {
			return {
				tabsOpen: tabsOpen
			};
		},
		restore: (value) => {
			tabsOpen = value.tabsOpen;
		}
	};

	let allTrainings = getTrainings();
	let allHolds = getHolds();

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

	async function getHolds(): Promise<Hold[]> {
		const holds = await target.getAllHolds(true, true);
		return holds.sort((a, b) => {
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

	let createHoldModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteTrainingModal: DeleteModalOptions = {
		open: false,
		name: '',
		onConfirm: async () => {}
	};

	let deleteHoldModal: DeleteModalOptions = {
		open: false,
		name: '',
		onConfirm: async () => {}
	};

	function createTraining() {
		createTrainingModal = {
			open: true,
			onConfirm: async () => {
				api.leashGet
				allTrainings = getTrainings();
			}
		};
	}

	function createHold() {
		createHoldModal = {
			open: true,
			onConfirm: async () => {allHolds = getHolds()}
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
				allTrainings = getTrainings();
			}
		};
	}

	function deleteHold(hold: Hold) {
		deleteHoldModal = {
			open: true,
			name: hold.holdType,
			onConfirm: async () => {
				deleteHoldModal.open = false;
				await hold.delete();
				await timeout(300);
				allHolds = getHolds();
			}
		};
	}
</script>

<svelte:head>
	<title>mkr.cx | Edit User</title>
</svelte:head>

<Tabs style="underline">
	<TabItem bind:open={tabsOpen.profile}>
		<div slot="title" class="flex items-center gap-2">
			<UserCircleSolid size="sm" />
			Profile
		</div>
		{#if target.role === 'service'}
			<ServiceProfileTab {data} />
		{:else}
			<UserProfileTab {data} />
		{/if}
	</TabItem>
	<TabItem bind:open={tabsOpen.trainings}>
		<div slot="title" class="flex items-center gap-2">
			<RectangleListSolid size="sm" />
			Trainings
		</div>

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

		<Button
			color="primary"
			class="mb-4 w-full"
			on:click={createTraining}
		>
			New Training
		</Button>
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
				{#await allTrainings}
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
			</TableBody>
		</Table>
	</TabItem>
	<TabItem bind:open={tabsOpen.holds}>
		<div slot="title" class="flex items-center gap-2">
			<AnnotationSolid size="sm" />
			Holds
		</div>

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
			onConfirm={deleteHoldModal.onConfirm}
		/>

		<Button
			color="primary"
			class="mb-4 w-full"
			on:click={createHold}
		>
			New Hold
		</Button>

		<Table>
			<TableHead>
				<TableHeadCell>Active</TableHeadCell>
				<TableHeadCell>Hold Type</TableHeadCell>
				<TableHeadCell>Reason</TableHeadCell>
				<TableHeadCell>Start Date</TableHeadCell>
				<TableHeadCell>End Date</TableHeadCell>
				<TableHeadCell>Date Added</TableHeadCell>
				<TableHeadCell>Added By</TableHeadCell>
				<TableHeadCell>Date Removed</TableHeadCell>
				<TableHeadCell>Removed By</TableHeadCell>
				<TableHeadCell>Remove</TableHeadCell>
			</TableHead>
			<TableBody>
				{#await allHolds}
					<TableBodyRow>
						<TableBodyCell colspan="10" class="p-0">Loading...</TableBodyCell>
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
							<TableBodyCell>{hold.holdType}</TableBodyCell>
							<TableBodyCell>{hold.reason}</TableBodyCell>
							<TableBodyCell>
								{#if hold.holdStart}
									<Timestamp timestamp={hold.holdStart} />
								{:else}
									-
								{/if}
							</TableBodyCell>
							<TableBodyCell>
								{#if hold.holdEnd}
									<Timestamp timestamp={hold.holdEnd} />
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
						<TableBodyCell colspan="10" class="p-0">Error: {error.message}</TableBodyCell>
					</TableBodyRow>
				{/await}
			</TableBody>
		</Table>
	</TabItem>
</Tabs>
