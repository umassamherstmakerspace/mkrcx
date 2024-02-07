<script lang="ts">
	import type { Hold, Training, User } from '$lib/leash';
	import {
		Button,
		CloseButton,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell,
		type IndicatorColorType
	} from 'flowbite-svelte';
	import { slide } from 'svelte/transition';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import { createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher();

	export let user: User;
	export let open: boolean;

	let userColor: IndicatorColorType = 'green';

	function updateUserColor(user: User) {
		user.getAllHolds().then((holds) => {
			if (holds.length === 0) {
				userColor = 'green';
				return;
			}

			const minHold = holds
				.filter((hold) => hold.isActive)
				.reduce((a, b) => (a.priority < b.priority ? a : b));

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
		<UserCell {user} dot={{ color: userColor }} />
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
								<TableBodyCell><Timestamp timestamp={user.createdAt} /></TableBodyCell>
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
								<TableBodyCell colspan="2" class="p-0">
									<Button color="yellow" class="w-full" href={'/staff/user/' + user.id}
										>Edit User</Button
									>
								</TableBodyCell>
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
									<TableBodyCell colspan="2" class="p-0">Loading...</TableBodyCell>
								</TableBodyRow>
							{:then trainings}
								{#each trainings as training}
									<TableBodyRow>
										<TableBodyCell>{training.trainingType}</TableBodyCell>
										<TableBodyCell
											><CloseButton on:click={() => deleteTraining(training)} /></TableBodyCell
										>
									</TableBodyRow>
								{/each}
							{:catch error}
								<TableBodyRow>
									<TableBodyCell colspan="2" class="p-0">Error: {error.message}</TableBodyCell>
								</TableBodyRow>
							{/await}
							<TableBodyRow>
								<TableBodyCell colspan="2" class="p-0">
									<Button color="green" on:click={() => createTraining()} class="w-full"
										>Add Training</Button
									>
								</TableBodyCell>
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
									<TableBodyCell colspan="6" class="p-0">Loading...</TableBodyCell>
								</TableBodyRow>
							{:then holds}
								{#each holds as hold}
									<TableBodyRow>
										<TableBodyCell>{hold.holdType}</TableBodyCell>
										<TableBodyCell>{hold.reason}</TableBodyCell>
										<TableBodyCell>{hold.priority}</TableBodyCell>
										<TableBodyCell>
											{#if hold.holdStart}
												<div class="flex flex-col">
													<Timestamp timestamp={hold.holdStart} formatter="date" />
													<Timestamp timestamp={hold.holdStart} formatter="time" />
												</div>
											{:else}
												<span></span>
											{/if}
										</TableBodyCell>
										<TableBodyCell>
											{#if hold.holdEnd}
												<div class="flex flex-col">
													<Timestamp timestamp={hold.holdEnd} formatter="date" />
													<Timestamp timestamp={hold.holdEnd} formatter="time" />
												</div>
											{:else}
												<span></span>
											{/if}
										</TableBodyCell>
										<TableBodyCell><CloseButton on:click={() => deleteHold(hold)} /></TableBodyCell>
									</TableBodyRow>
								{/each}
							{:catch error}
								<TableBodyRow>
									<TableBodyCell colspan="6" class="p-0">Error: {error.message}</TableBodyCell>
								</TableBodyRow>
							{/await}
							<TableBodyRow>
								<TableBodyCell colspan="6" class="p-0">
									<Button color="green" on:click={() => createHold()} class="w-full"
										>Add Hold</Button
									>
								</TableBodyCell>
							</TableBodyRow>
						</TableBody>
					</Table>
				</div>
			</div>
		</TableBodyCell>
	</TableBodyRow>
{/if}
