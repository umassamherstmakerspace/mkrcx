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
		Indicator
	} from 'flowbite-svelte';
	import { RectangleListSolid, AnnotationSolid, UserCircleSolid } from 'flowbite-svelte-icons';
	import UserCell from '$lib/components/UserCell.svelte';

	import type { PageData, Snapshot } from './$types';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserProfileTab from './UserProfileTab.svelte';
	import ServiceProfileTab from './ServiceProfileTab.svelte';

	export let data: PageData;
	const { user, tabs, tabsOpen } = data;
	let { target } = data;

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

	async function getHolds() {
		const holds = await target.getAllHolds();
		return holds.filter((hold) => {
			if (hold.holdEnd == undefined) return true;
			return hold.holdEnd.getTime() > Date.now();
		});
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
		<Table>
			<TableHead>
				<TableHeadCell>Active</TableHeadCell>
				<TableHeadCell>Training Type</TableHeadCell>
				<TableHeadCell>Date Added</TableHeadCell>
				<TableHeadCell>Added By</TableHeadCell>
				<TableHeadCell>Date Removed</TableHeadCell>
				<TableHeadCell>Removed By</TableHeadCell>
			</TableHead>
			<TableBody>
				{#await target.getAllTrainings()}
					<TableBodyRow>
						<TableBodyCell colspan="2" class="p-0">Loading...</TableBodyCell>
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
						</TableBodyRow>
					{/each}
				{:catch error}
					<TableBodyRow>
						<TableBodyCell colspan="2" class="p-0">Error: {error.message}</TableBodyCell>
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
			</TableHead>
			<TableBody>
				{#await getHolds()}
					<TableBodyRow>
						<TableBodyCell colspan="2" class="p-0">Loading...</TableBodyCell>
					</TableBodyRow>
				{:then holds}
					{#each holds as hold}
						<TableBodyRow>
							<TableBodyCell>
								{#if hold.isActive() || hold.deletedAt}
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
						</TableBodyRow>
					{/each}
				{:catch error}
					<TableBodyRow>
						<TableBodyCell colspan="2" class="p-0">Error: {error.message}</TableBodyCell>
					</TableBodyRow>
				{/await}
			</TableBody>
		</Table>
	</TabItem>
</Tabs>
