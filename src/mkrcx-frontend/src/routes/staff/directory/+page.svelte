<script lang="ts">
	import UserRow from "$lib/components/UserRow.svelte";
	import { Hold, Training, User } from "$lib/leash";
	import { Button, Input, Label, Modal, NumberInput, P, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, TableSearch } from "flowbite-svelte";
	import { ExclamationCircleOutline } from "flowbite-svelte-icons";
    import { inview, type Options as InviewOptions } from 'svelte-inview';
    import { getUnixTime } from "date-fns";

    let query = "";

    let users: User[] = [];
    let offset = 0;
    let hasMore = true;
    let loaded = false;

    const inviewOptions: InviewOptions = {
    rootMargin: '50px',
  };

    async function search(q: string) {
        if (!hasMore) return;
        loaded = false;

        const res = await User.search(q, {
            offset,
            limit: 50,
            withHolds: true,
            withTrainings: true,
        });

        users = [...users, ...res.data];
        offset += res.data.length;
        hasMore = res.total > offset;
        loaded = true;
    }

    function newSearch(q: string) {
        loaded = false;
        offset = 0;
        users = [];
        hasMore = true;
        search(q);
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
    };

    interface CreateHoldModalOptions extends ModalOptions {
        reason: string;
        priority: number;
        startDate?: Date;
        endDate?: Date;
    }

    let deleteTrainingModal: ModalOptions = {
        open: false,
        type: "",
        targetUser: null,
        onConfirm: () => {},
    };

    let deleteHoldModal: ModalOptions = {
        open: false,
        type: "",
        targetUser: null,
        onConfirm: () => {},
    };

    let createHoldModal: CreateHoldModalOptions = {
        open: false,
        type: "",
        targetUser: null,
        reason: "",
        priority: 0,
        startDate: undefined,
        endDate: undefined,
        onConfirm: () => {},
    };

    let createTrainingModal: ModalOptions = {
        open: false,
        type: "",
        targetUser: null,
        onConfirm: () => {},
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
            },
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
            },
        };
    }

    async function createTraining() {
        if (activeRow === null) return;

        createTrainingModal = {
            open: true,
            type: "",
            targetUser: users[activeRow],
            onConfirm: async () => {
                createTrainingModal.open = false;
                if (activeRow === null) return;
                await users[activeRow].createTraining({
                    trainingType: createTrainingModal.type,
                });
                await reloadUser();
            },
        };
    }

    async function createHold() {
        if (activeRow === null) return;

        createHoldModal = {
            open: true,
            type: "",
            targetUser: users[activeRow],
            reason: "",
            priority: 0,
            startDate: undefined,
            endDate: undefined,
            onConfirm: async () => {
                createHoldModal.open = false;
                if (activeRow === null) return;
                const holdStart = createHoldModal.startDate ? getUnixTime(createHoldModal.startDate) : undefined;
                const holdEnd = createHoldModal.endDate ? getUnixTime(createHoldModal.endDate) : undefined;

                await users[activeRow].createHold({
                    holdType: createHoldModal.type,
                    reason: createHoldModal.reason,
                    priority: createHoldModal.priority,
                    holdStart,
                    holdEnd,
                });
                await reloadUser();
            },
        };
    }
</script>

<Modal bind:open={deleteTrainingModal.open} size="xs" autoclose>
    <div class="text-center">
      <ExclamationCircleOutline class="mx-auto mb-4 text-gray-400 w-12 h-12 dark:text-gray-200" />
      <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">Are you sure you want to remove the {deleteTrainingModal.type} training from {deleteTrainingModal.targetUser?.name || "error"}?</h3>
      <Button color="red" class="me-2" on:click={deleteTrainingModal.onConfirm}>Remove Training</Button>
      <Button color="alternative" on:click={() => deleteTrainingModal.open = false}>Cancel</Button>
    </div>
  </Modal>

  <Modal bind:open={deleteHoldModal.open} size="xs" autoclose>
    <div class="text-center">
      <ExclamationCircleOutline class="mx-auto mb-4 text-gray-400 w-12 h-12 dark:text-gray-200" />
      <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">Are you sure you want to remove the {deleteHoldModal.type} hold from {deleteHoldModal.targetUser?.name || "error"}?</h3>
      <Button color="red" class="me-2" on:click={deleteHoldModal.onConfirm}>Remove Hold</Button>
      <Button color="alternative" on:click={() => deleteHoldModal.open = false}>Cancel</Button>
    </div>
  </Modal>


<Modal bind:open={createTrainingModal.open} size="xs" autoclose={false} class="w-full">
    <div class="flex flex-col space-y-6">
      <h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">Create training for {createTrainingModal.targetUser?.name || "error"}</h3>
      <Label class="space-y-2">
        <span>Training Type</span>
        <Input type="text" name="text" placeholder="Training Type" required bind:value={createTrainingModal.type} />
      </Label>
      <Button class="w-full1" on:click={createTrainingModal.onConfirm}>Create Training</Button>
    </div>
  </Modal>

  <Modal bind:open={createHoldModal.open} size="xs" autoclose={false} class="w-full">
    <div class="flex flex-col space-y-6">
      <h3 class="mb-4 text-xl font-medium text-gray-900 dark:text-white">Create hold for {createHoldModal.targetUser?.name || "error"}</h3>
      <Label class="space-y-2">
        <span>Hold Type</span>
        <Input type="text" name="text" placeholder="Hold Type" required bind:value={createHoldModal.type} />
      </Label>
      <Label class="space-y-2">
        <span>Reason</span>
        <Input type="text" name="text" placeholder="Reason" required bind:value={createHoldModal.reason} />
      </Label>
      <Label class="space-y-2">
        <span>Priority</span>
        <NumberInput type="number" name="text" placeholder="Priority" required bind:value={createHoldModal.priority} />
      </Label>
      <Label class="space-y-2">
        <span>Start Date</span>
        <Input type="date" name="text" placeholder="Start Date" required bind:value={createHoldModal.startDate} />
      </Label>
      <Label class="space-y-2">
        <span>End Date</span>
        <Input type="date" name="text" placeholder="End Date" required bind:value={createHoldModal.endDate} />
      </Label>
      <Button class="w-full1" on:click={createHoldModal.onConfirm}>Create Hold</Button>
    </div>
    </Modal>
<div class="flex flex-col p-4 w-full">
    <TableSearch placeholder="Search by name or email..." hoverable={true} bind:inputValue={query} >
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
            <UserRow user={user} open={activeRow === i} on:click={() => toggleRow(i)} on:deleteHold={deleteHold} on:deleteTraining={deleteTraining} on:createHold={createHold} on:createTraining={createTraining} />
          {/each}
          {#if loaded && hasMore}
          <TableBodyRow>
            <TableBodyCell colspan="8" class="p-0">
              <div class="px-2 py-3" use:inview={inviewOptions} on:inview_enter={() => search(query)}>
                <div class="animate-pulse flex flex-col items-center w-full">
                    <P size="sm" weight="light" class="text-gray-200 dark:text-gray-700 mb-2.5"> Loading More... </P>
                    <div class="h-2 bg-gray-200 rounded-full dark:bg-gray-700 mb-2.5 w-full" />
                    <div class="h-2 bg-gray-200 rounded-full dark:bg-gray-700 mb-2.5 w-full" />
                </div>
              </div>
            </TableBodyCell>
        </TableBodyRow>
        {/if}
        </TableBody>
    </Table>
      </TableSearch>
</div>