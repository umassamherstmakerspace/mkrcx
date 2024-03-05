<script lang="ts">
	import {
		Tabs,
		TabItem,
	} from 'flowbite-svelte';
	import { RectangleListSolid, AnnotationSolid, UserCircleSolid, LockSolid, TerminalSolid, BellSolid, UserEditSolid } from 'flowbite-svelte-icons';

	import type { PageData, Snapshot } from './$types';
	import UserProfileTab from './UserProfileTab.svelte';
	import ServiceProfileTab from './ServiceProfileTab.svelte';
	import TrainingTab from '../TrainingTab.svelte';
	import HoldTab from '../HoldTab.svelte';
	// import NoficationTab from '../NoficationTab.svelte';
	import UserUpdateTab from '../UserUpdateTab.svelte';
	import ApikeyTab from '../ApikeyTab.svelte';
	import AdminTab from '../AdminTab.svelte';

	export let data: PageData;
	const { tabs, user } = data;
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
</script>

<svelte:head>
	<title>mkr.cx | Edit User</title>
</svelte:head>

<Tabs style="underline">
	{#if tabs.profile}
		<TabItem bind:open={tabsOpen.profile}>
			<div slot="title" class="flex items-center gap-2">
				<UserCircleSolid size="sm" />
				Profile
			</div>
			{#if target.role === 'service'}
				<ServiceProfileTab bind:target />
			{:else}
				<UserProfileTab bind:target {user} />
			{/if}
		</TabItem>
	{/if}
	{#if tabs.trainings}
		<TabItem bind:open={tabsOpen.trainings}>
			<div slot="title" class="flex items-center gap-2">
				<RectangleListSolid size="sm" />
				Trainings
			</div>

			<TrainingTab {target} />
		</TabItem>
	{/if}
	{#if tabs.holds}
		<TabItem bind:open={tabsOpen.holds}>
			<div slot="title" class="flex items-center gap-2">
				<AnnotationSolid size="sm" />
				Holds
			</div>

			<HoldTab {target} />
		</TabItem>
	{/if}
	{#if tabs.notifications}
		<TabItem bind:open={tabsOpen.notifications}>
			<div slot="title" class="flex items-center gap-2">
				<BellSolid size="sm" />
				Nofications
			</div>

			<!-- <NoficationTab {target} /> -->
		</TabItem>
	{/if}
	{#if tabs.updates}
		<TabItem bind:open={tabsOpen.updates}>
			<div slot="title" class="flex items-center gap-2">
				<UserEditSolid size="sm" />
				User Updates
			</div>

			<UserUpdateTab {target} />
		</TabItem>
	{/if}
	{#if tabs.apikeys}
		<TabItem bind:open={tabsOpen.apikeys}>
			<div slot="title" class="flex items-center gap-2">
				<LockSolid size="sm" />
				Api Keys
			</div>

			<ApikeyTab {target} />
		</TabItem>
	{/if}
	{#if tabs.admin}
		<TabItem bind:open={tabsOpen.admin}>
			<div slot="title" class="flex items-center gap-2">
				<TerminalSolid size="sm" />
				Admin Page
			</div>

			<AdminTab {target} />
		</TabItem>
	{/if}
</Tabs>
