<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import CreateApikeyModal from '$lib/components/modals/CreateApikeyModal.svelte';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import type { ModalOptions } from '$lib/components/modals/modals';
	import type { APIKey, User } from '$lib/leash';
	import {
		Badge,
		Button,
		CloseButton,
		Indicator,
		Modal,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';
	import { FileSearchOutline } from 'flowbite-svelte-icons';

	export let target: User;

	let apikeys = {};

	async function getAPIKeys(): Promise<APIKey[]> {
		const apikeys = await target.getAllAPIKeys(true, true);
		return apikeys.sort((a, b) => {
			if (a.deletedAt && b.deletedAt) return b.deletedAt.getTime() - a.deletedAt.getTime();
			if (a.deletedAt) return 1;
			if (b.deletedAt) return -1;
			else return b.createdAt.getTime() - a.createdAt.getTime();
		});
	}

	let createAPIKeyModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteAPIKeyModal: DeleteModalOptions = {
		open: false,
		name: '',
		deleteFn: async () => {},
		onConfirm: async () => {}
	};

	function createApikey() {
		createAPIKeyModal = {
			open: true,
			onConfirm: async () => {
				apikeys = {};
			}
		};
	}

	function deleteApikey(apikey: APIKey) {
		deleteAPIKeyModal = {
			open: true,
			name: apikey.key,
			deleteFn: apikey.delete,
			onConfirm: async () => {
				apikeys = {};
			}
		};
	}

	let showKey: APIKey | null = null;
</script>

<Modal
	open={showKey != null}
	on:close={() => (showKey = null)}
	size="md"
	autoclose={false}
	class="w-full"
>
	{#if showKey}
		<div class="text-center">
			<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
				Permissions for {showKey.key}
			</h3>
			<div class="text-left">
				{#each showKey.permissions as permission}
					<p class="mb-2">{permission}</p>
				{/each}
			</div>
			<Button color="primary" class="mt-5 w-full" on:click={() => (showKey = null)}>Close</Button>
		</div>
	{/if}
</Modal>

<CreateApikeyModal
	bind:open={createAPIKeyModal.open}
	user={target}
	onConfirm={createAPIKeyModal.onConfirm}
/>

<DeleteModal
	bind:open={deleteAPIKeyModal.open}
	modalType="API Key"
	name={deleteAPIKeyModal.name}
	user={target}
	deleteFn={deleteAPIKeyModal.deleteFn}
	onConfirm={deleteAPIKeyModal.onConfirm}
/>

<Button color="primary" class="mb-4 w-full" on:click={createApikey}>New API Key</Button>

<Table>
	<TableHead>
		<TableHeadCell>Active</TableHeadCell>
		<TableHeadCell>Key</TableHeadCell>
		<TableHeadCell>Description</TableHeadCell>
		<TableHeadCell>Permissions</TableHeadCell>
		<TableHeadCell>Date Added</TableHeadCell>
		<TableHeadCell>Date Removed</TableHeadCell>
		<TableHeadCell>Remove</TableHeadCell>
	</TableHead>
	<TableBody>
		{#key apikeys}
			{#await getAPIKeys()}
				<TableBodyRow>
					<TableBodyCell colspan="7" class="p-0">Loading...</TableBodyCell>
				</TableBodyRow>
			{:then apikeys}
				{#each apikeys as apikey}
					<TableBodyRow>
						<TableBodyCell>
							{#if apikey.deletedAt}
								<Badge color="red" rounded class="px-2.5 py-0.5">
									<Indicator color="red" size="xs" class="me-1" />Deleted
								</Badge>
							{:else}
								<Badge color="green" rounded class="px-2.5 py-0.5">
									<Indicator color="green" size="xs" class="me-1" />Active
								</Badge>
							{/if}
						</TableBodyCell>
						<TableBodyCell>{apikey.key}</TableBodyCell>
						<TableBodyCell>{apikey.description}</TableBodyCell>
						<TableBodyCell>
							<Button
								class="!p-2"
								color="none"
								size="sm"
								on:click={() => {
									showKey = apikey;
								}}
							>
								<FileSearchOutline class="h-5 w-5" />
							</Button>
						</TableBodyCell>
						<TableBodyCell><Timestamp timestamp={apikey.createdAt} /></TableBodyCell>
						<TableBodyCell>
							{#if apikey.deletedAt}
								<Timestamp timestamp={apikey.deletedAt} />
							{:else}
								-
							{/if}
						</TableBodyCell>
						<TableBodyCell>
							{#if !apikey.deletedAt}
								<CloseButton on:click={() => deleteApikey(apikey)} />
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
