<script lang="ts">
	import UserRow from './UserRow.svelte';
	import { Hold, Training, User } from '$lib/leash';
	import {
		Button,
		Input,
		Label,
		Modal,
		NumberInput,
		P,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell,
		TableSearch
	} from 'flowbite-svelte';
	import { ExclamationCircleOutline } from 'flowbite-svelte-icons';
	import { inview, type Options as InviewOptions } from 'svelte-inview';
	import { getUnixTime } from 'date-fns';
	import type { Snapshot } from './$types';

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

		const res = await User.search(q, {
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

	function timeout(ms: number) {
		return new Promise((resolve) => setTimeout(resolve, ms));
	}

	async function reloadUser() {
		if (activeRow === null) return;
		users[activeRow] = await users[activeRow].get({ withHolds: true, withTrainings: true });
		users = [...users];
	}

	function toggleRow(i: number) {
		activeRow = activeRow === i ? null : i;
	}

	interface ModalOptions {
		open: boolean;
		type: string;
		targetUser: User | null;
		onConfirm: () => void;
	}

	interface CreateHoldModalOptions extends ModalOptions {
		reason: string;
		priority: number;
		startDate?: Date;
		endDate?: Date;
	}

	let deleteTrainingModal: ModalOptions = {
		open: false,
		type: '',
		targetUser: null,
		onConfirm: () => {}
	};

	let deleteHoldModal: ModalOptions = {
		open: false,
		type: '',
		targetUser: null,
		onConfirm: () => {}
	};

	let createHoldModal: CreateHoldModalOptions = {
		open: false,
		type: '',
		targetUser: null,
		reason: '',
		priority: 0,
		startDate: undefined,
		endDate: undefined,
		onConfirm: () => {}
	};

	$: if (createHoldModal.priority < 0) createHoldModal.priority = 0;

	let createTrainingModal: ModalOptions = {
		open: false,
		type: '',
		targetUser: null,
		onConfirm: () => {}
	};

	function deleteTraining(event: CustomEvent<Training>) {
		if (activeRow === null) return;
		deleteTrainingModal = {
			open: true,
			type: event.detail.trainingType,
			targetUser: users[activeRow],
			onConfirm: async () => {
				deleteTrainingModal.open = false;
				await event.detail.delete();
				await timeout(300);
				await reloadUser();
			}
		};
	}

	function deleteHold(event: CustomEvent<Hold>) {
		if (activeRow === null) return;
		deleteHoldModal = {
			open: true,
			type: event.detail.holdType,
			targetUser: users[activeRow],
			onConfirm: async () => {
				deleteHoldModal.open = false;
				await event.detail.delete();
				await timeout(300);
				await reloadUser();
			}
		};
	}

	async function createTraining() {
		if (activeRow === null) return;

		createTrainingModal = {
			open: true,
			type: '',
			targetUser: users[activeRow],
			onConfirm: async () => {
				createTrainingModal.open = false;
				if (activeRow === null) return;
				await users[activeRow].createTraining({
					trainingType: createTrainingModal.type
				});
				await reloadUser();
			}
		};
	}

	async function createHold() {
		if (activeRow === null) return;

		createHoldModal = {
			open: true,
			type: '',
			targetUser: users[activeRow],
			reason: '',
			priority: 0,
			startDate: undefined,
			endDate: undefined,
			onConfirm: async () => {
				createHoldModal.open = false;
				if (activeRow === null) return;
				const holdStart = createHoldModal.startDate
					? getUnixTime(createHoldModal.startDate)
					: undefined;
				const holdEnd = createHoldModal.endDate ? getUnixTime(createHoldModal.endDate) : undefined;

				await users[activeRow].createHold({
					holdType: createHoldModal.type,
					reason: createHoldModal.reason,
					priority: createHoldModal.priority,
					holdStart,
					holdEnd
				});
				await reloadUser();
			}
		};
	}
</script>

<Modal bind:open={deleteTrainingModal.open} size="xs" autoclose>
	<div class="text-center">
		<ExclamationCircleOutline class="mx-auto mb-4 h-12 w-12 text-gray-400 dark:text-gray-200" />
		<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
			Are you sure you want to remove the {deleteTrainingModal.type} training from {deleteTrainingModal
				.targetUser?.name || 'error'}?
		</h3>
		<Button color="red" class="me-2" on:click={deleteTrainingModal.onConfirm}
			>Remove Training</Button
		>
		<Button color="alternative" on:click={() => (deleteTrainingModal.open = false)}>Cancel</Button>
	</div>
</Modal>

<Modal bind:open={deleteHoldModal.open} size="xs" autoclose>
	<div class="text-center">
		<ExclamationCircleOutline class="mx-auto mb-4 h-12 w-12 text-gray-400 dark:text-gray-200" />
		<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
			Are you sure you want to remove the {deleteHoldModal.type} hold from {deleteHoldModal
				.targetUser?.name || 'error'}?
		</h3>
		<Button color="red" class="me-2" on:click={deleteHoldModal.onConfirm}>Remove Hold</Button>
		<Button color="alternative" on:click={() => (deleteHoldModal.open = false)}>Cancel</Button>
	</div>
</Modal>

<Modal bind:open={createTrainingModal.open} size="xs" autoclose={false} class="w-full">
	<form
		class="flex flex-col space-y-6"
		method="dialog"
		on:submit|preventDefault={createTrainingModal.onConfirm}
	>
		<h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">
			Create training for {createTrainingModal.targetUser?.name || 'error'}
		</h3>
		<Label class="space-y-2" for="training-type">
			<span>Training Type</span>
			<Input
				type="text"
				name="training-type"
				placeholder="Training Type"
				required
				bind:value={createTrainingModal.type}
			/>
		</Label>
		<Button class="w-full1" type="submit">Create Training</Button>
	</form>
</Modal>

<Modal bind:open={createHoldModal.open} size="xs" autoclose={false} class="w-full">
	<form
		class="flex flex-col space-y-6"
		method="dialog"
		on:submit|preventDefault={createHoldModal.onConfirm}
	>
		<h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">
			Create hold for {createHoldModal.targetUser?.name || 'error'}
		</h3>
		<Label class="space-y-2">
			<span>Hold Type</span>
			<Input
				type="text"
				name="text"
				placeholder="Hold Type"
				required
				bind:value={createHoldModal.type}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Reason</span>
			<Input
				type="text"
				name="text"
				placeholder="Reason"
				required
				bind:value={createHoldModal.reason}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Priority</span>
			<NumberInput
				type="number"
				name="text"
				placeholder="Priority"
				required
				bind:value={createHoldModal.priority}
			/>
		</Label>
		<Label class="space-y-2">
			<span>Start Date</span>
			<Input
				type="datetime-local"
				name="text"
				placeholder="Start Date"
				bind:value={createHoldModal.startDate}
			/>
		</Label>
		<Label class="space-y-2">
			<span>End Date</span>
			<Input
				type="datetime-local"
				name="text"
				placeholder="End Date"
				bind:value={createHoldModal.endDate}
			/>
		</Label>
		<Button class="w-full1" type="submit">Create Hold</Button>
	</form>
</Modal>
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
