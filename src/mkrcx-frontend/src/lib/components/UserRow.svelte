<script lang="ts">
	import type { Hold, Training, User } from '$lib/leash';
	import {
		Avatar,
		Button,
		CloseButton,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';
	import { slide } from 'svelte/transition';
	import Timestamp from './Timestamp.svelte';
    import { createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher();

	export let user: User;
	export let open: boolean;

	let userColor = 'green';

	function updateUserColor(user: User) {
		user.getAllHolds().then((holds) => {
			if (holds.length === 0) {
				userColor = 'green';
				return;
			}

			const minHold = holds.reduce((a, b) => (a.priority < b.priority ? a : b));

			if (minHold.priority < 100) {
				userColor = 'red';
			} else {
				userColor = 'yellow';
			}
		});
	}

	$: updateUserColor(user);

    function deleteHold(hold: Hold) {
        dispatch('deleteHold', hold);
    }

    function deleteTraining(training: Training) {
        dispatch('deleteTraining', training);
    }

    function createHold() {
        dispatch('createHold');
    }

    function createTraining() {
        dispatch('createTraining');
    }
</script>

<TableBodyRow on:click>
	<TableBodyCell>
		<div class="flex items-center">
			<Avatar id="avatar-menu" src={user.iconURL} rounded dot={{ color: userColor }} />
			<div class="ms-3">
				<div class="font-semibold">{user.name}</div>
				<div class="text-gray-500 dark:text-gray-400">{user.email}</div>
			</div>
		</div>
	</TableBodyCell>
	<TableBodyCell>{user.role}</TableBodyCell>
	<TableBodyCell>{user.type}</TableBodyCell>
	<TableBodyCell>{user.major}</TableBodyCell>
	{#if user.graduationYear > 0}
		<TableBodyCell>{user.graduationYear}</TableBodyCell>
	{:else}
		<TableBodyCell></TableBodyCell>
	{/if}
</TableBodyRow>
{#if open}
	<TableBodyRow>
		<TableBodyCell colspan={8} class="p-0">
			<div class="px-2 py-3" transition:slide={{ duration: 300, axis: 'y' }}>
				<div class="flex w-full gap-4">
                    <Table>
						<TableHead>
							<TableHeadCell>Field</TableHeadCell>
							<TableHeadCell>Value</TableHeadCell>
						</TableHead>
						<TableBody>
                            <TableBodyRow>
                                <TableBodyCell>ID</TableBodyCell>
                                <TableBodyCell>{user.id}</TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Join Date</TableBodyCell>
                                <TableBodyCell><Timestamp timestamp={user.createdAt}/></TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Name</TableBodyCell>
                                <TableBodyCell>{user.name}</TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Email</TableBodyCell>
                                <TableBodyCell>{user.email}</TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Pending Email</TableBodyCell>
                                {#if user.pendingEmail}
                                    <TableBodyCell>{user.pendingEmail}</TableBodyCell>
                                {:else}
                                    <TableBodyCell></TableBodyCell>
                                {/if}
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Role</TableBodyCell>
                                <TableBodyCell>{user.role}</TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Type</TableBodyCell>
                                <TableBodyCell>{user.type}</TableBodyCell>
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Major</TableBodyCell>
                                {#if user.major}
                                    <TableBodyCell>{user.major}</TableBodyCell>
                                {:else}
                                    <TableBodyCell></TableBodyCell>
                                {/if}
                            </TableBodyRow>
                            <TableBodyRow>
                                <TableBodyCell>Graduation Year</TableBodyCell>
                                {#if user.graduationYear > 0}
                                    <TableBodyCell>{user.graduationYear}</TableBodyCell>
                                {:else}
                                    <TableBodyCell></TableBodyCell>
                                {/if}
                            </TableBodyRow>
							<TableBodyRow>
								<Button color="yellow">Edit User</Button>
							</TableBodyRow>
						</TableBody>
					</Table>

					<Table>
						<TableHead>
							<TableHeadCell>Training Type</TableHeadCell>
							<TableHeadCell>Remove</TableHeadCell>
						</TableHead>
						<TableBody>
                            {#await user.getAllTrainings()}
                            <TableBodyRow>
                                <TableBodyCell>Loading...</TableBodyCell>
                            </TableBodyRow>
                            {:then trainings}
							{#each trainings as training}
								<TableBodyRow>
									<TableBodyCell>{training.trainingType}</TableBodyCell>
									<TableBodyCell><CloseButton on:click={() => deleteTraining(training)}/></TableBodyCell>
								</TableBodyRow>
							{/each}
                            {:catch error}
                            <TableBodyRow>
                                <TableBodyCell>Error: {error.message}</TableBodyCell>
                            </TableBodyRow>
                            {/await}
							<TableBodyRow>
								<Button color="green" on:click={() => createTraining()}>Add Training</Button>
							</TableBodyRow>
						</TableBody>
					</Table>

                    <Table>
						<TableHead>
							<TableHeadCell>Hold Type</TableHeadCell>
							<TableHeadCell>Reason</TableHeadCell>
                            <TableHeadCell>Priority</TableHeadCell>
							<TableHeadCell>Start Time</TableHeadCell>
							<TableHeadCell>End Time</TableHeadCell>
							<TableHeadCell>Remove</TableHeadCell>
						</TableHead>
						<TableBody>
                            {#await user.getAllHolds()}
                            <TableBodyRow>
                                <TableBodyCell>Loading...</TableBodyCell>
                            </TableBodyRow>
                            {:then holds}
							{#each holds as hold}
								<TableBodyRow>
									<TableBodyCell>{hold.holdType}</TableBodyCell>
                                    <TableBodyCell>{hold.reason}</TableBodyCell>
                                    <TableBodyCell>{hold.priority}</TableBodyCell>
                                    <TableBodyCell>
                                        {#if hold.holdStart}
                                            <Timestamp timestamp={hold.holdStart}/>
                                        {:else}
                                            <span></span>
                                        {/if}
                                    </TableBodyCell>
                                    <TableBodyCell>
                                        {#if hold.holdEnd}
                                            <Timestamp timestamp={hold.holdEnd}/>
                                        {:else}
                                            <span></span>
                                        {/if}
                                    </TableBodyCell>
									<TableBodyCell><CloseButton on:click={() => deleteHold(hold)}/></TableBodyCell>
								</TableBodyRow>
							{/each}
                            {:catch error}
                            <TableBodyRow>
                                <TableBodyCell>Error: {error.message}</TableBodyCell>
                            </TableBodyRow>
                            {/await}
							<TableBodyRow>
								<Button color="green" on:click={() => createHold()}>Add Hold</Button>
							</TableBodyRow>
						</TableBody>
					</Table>
				</div>
			</div>
		</TableBodyCell>
	</TableBodyRow>
{/if}