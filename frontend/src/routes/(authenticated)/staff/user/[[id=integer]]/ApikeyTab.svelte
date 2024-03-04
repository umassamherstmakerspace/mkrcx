<!-- <script lang="ts">
	import Timestamp from "$lib/components/Timestamp.svelte";
	import UserCell from "$lib/components/UserCell.svelte";
	import CreateHoldModal from "$lib/components/modals/CreateHoldModal.svelte";
	import DeleteModal, { type DeleteModalOptions } from "$lib/components/modals/DeleteModal.svelte";
	import { timeout, type ModalOptions } from "$lib/components/modals/modals";
	import type { APIKey, User } from "$lib/leash";
	import { Badge, Button, CloseButton, Indicator, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell } from "flowbite-svelte";

    export let target: User;

    let apikeys = {};

    async function getAPIKeys(): Promise<APIKey[]> {
		const apikeys = await target.getAllAPIKeys(true, true);
		return apikeys.sort((a, b) => {
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

	let createAPIKeyModal: ModalOptions = {
		open: false,
		onConfirm: async () => {}
	};

	let deleteAPIKeyModal: DeleteModalOptions = {
		open: false,
		name: '',
		onConfirm: async () => {}
	};

	function createApikey() {
		createAPIKeyModal = {
			open: true,
			onConfirm: async () => {apikeys = {}}
		};
	}


	function deleteApikey(apikey: APIKey) {
		deleteAPIKeyModal = {
			open: true,
			name: apikey.key,
			onConfirm: async () => {
				deleteAPIKeyModal.open = false;
				await apikey.delete();
				await timeout(300);
				apikeys = {};
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
                {#key holds}
                    {#await getHolds()}
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
                {/key}
			</TableBody>
		</Table> -->