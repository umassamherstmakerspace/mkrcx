<script lang="ts">
	import UserRow from './UserRow.svelte';
	import { Hold, Training, User } from '$lib/leash';
	import {
		P,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell,
		TableSearch
	} from 'flowbite-svelte';
	import { inview, type Options as InviewOptions } from 'svelte-inview';
	import type { PageData, Snapshot } from './$types';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import { timeout, type ModalOptions } from '$lib/components/modals/modals';
	import CreateTrainingModal from '$lib/components/modals/CreateTrainingModal.svelte';
	import CreateHoldModal from '$lib/components/modals/CreateHoldModal.svelte';

	export let data: PageData;
	let { api } = data;

	type Data = {
		query: string;
	};

	let lastRequestTime = 0;
	let query: string = '';

	export const snapshot: Snapshot<Data> = {
		capture: () => {
			return {
				query: query
			};
		},
		restore: (value) => {
			query = value.query;
		}
	};

	let users: User[] = [];
	let offset = 0;
	let loadResult = Promise.resolve(false);

	const inviewOptions: InviewOptions = {
		rootMargin: '50px'
	};

	async function search(q: string, list: User[] = [], requestOffset = 0): Promise<boolean> {
		const now = Date.now();

		const res = await api.searchUsers(q, {
			offset: requestOffset,
			limit: 50,
			withHolds: true,
			withTrainings: true
		});

		if (lastRequestTime >= now) return false;
		lastRequestTime = now;

		users = [...list, ...res.data];
		requestOffset += res.data.length;
		offset = requestOffset;
		return res.total > requestOffset;
	}

	function newSearch(query: string) {
		activeRow = null;
		loadResult = search(query);
	}

	$: newSearch(query);

	let activeRow: number | null = null;

	async function reloadUser() {
		if (activeRow === null) return;
		users[activeRow] = await users[activeRow].get({ withHolds: true, withTrainings: true });
		users = [...users];
	}

	function toggleRow(i: number) {
		activeRow = activeRow === i ? null : i;
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
			onConfirm: reloadUser
		};
	}

	function createHold() {
		createHoldModal = {
			open: true,
			onConfirm: reloadUser
		};
	}

	function deleteTraining(event: CustomEvent<Training>) {
		deleteTrainingModal = {
			open: true,
			name: event.detail.trainingType,
			onConfirm: async () => {
				deleteTrainingModal.open = false;
				await event.detail.delete();
				await timeout(300);
				await reloadUser();
			}
		};
	}

	function deleteHold(event: CustomEvent<Hold>) {
		deleteHoldModal = {
			open: true,
			name: event.detail.holdType,
			onConfirm: async () => {
				deleteHoldModal.open = false;
				await event.detail.delete();
				await timeout(300);
				await reloadUser();
			}
		};
	}
</script>

{#if activeRow !== null}
	{@const user = users[activeRow]}
	<CreateTrainingModal
		bind:open={createTrainingModal.open}
		{user}
		onConfirm={createTrainingModal.onConfirm}
	/>

	<CreateHoldModal bind:open={createHoldModal.open} {user} onConfirm={createHoldModal.onConfirm} />

	<DeleteModal
		bind:open={deleteTrainingModal.open}
		modalType="Training"
		name={deleteTrainingModal.name}
		{user}
		onConfirm={deleteTrainingModal.onConfirm}
	/>

	<DeleteModal
		bind:open={deleteHoldModal.open}
		modalType="Hold"
		name={deleteHoldModal.name}
		{user}
		onConfirm={deleteHoldModal.onConfirm}
	/>
{/if}

<div class="flex w-full flex-col">
	<TableSearch placeholder="Search by name or email..." hoverable={true} bind:inputValue={query}>
		<Table divClass="relative overflow-x-auto overflow-y-auto max-h-fit">
			<TableHead>
				<TableHeadCell>Name</TableHeadCell>
				<TableHeadCell>Role</TableHeadCell>
				<TableHeadCell>Type</TableHeadCell>
				<TableHeadCell>Major</TableHeadCell>
				<TableHeadCell>Graduation Year</TableHeadCell>
			</TableHead>
			<TableBody tableBodyClass="divide-y divide-gray-200 dark:divide-gray-700">
				{#each users as user, i}
					<UserRow
						{user}
						open={activeRow === i}
						on:click={() => toggleRow(i)}
						on:deleteHold={deleteHold}
						on:deleteTraining={deleteTraining}
						on:createHold={createHold}
						on:createTraining={createTraining}
					/>
				{/each}
				{#await loadResult then hasMore}
					{#if hasMore}
						<TableBodyRow>
							<TableBodyCell colspan="8" class="p-0">
								<div
									class="px-2 py-3"
									use:inview={inviewOptions}
									on:inview_enter={() => (loadResult = search(query, users, offset))}
								>
									<div class="flex w-full animate-pulse flex-col items-center">
										<P size="sm" weight="light" class="mb-2.5 text-gray-200 dark:text-gray-700">
											Loading More...
										</P>
										<div class="mb-2.5 h-2 w-full rounded-full bg-gray-200 dark:bg-gray-700" />
										<div class="mb-2.5 h-2 w-full rounded-full bg-gray-200 dark:bg-gray-700" />
									</div>
								</div>
							</TableBodyCell>
						</TableBodyRow>
					{/if}
				{/await}
			</TableBody>
		</Table>
	</TableSearch>
</div>
