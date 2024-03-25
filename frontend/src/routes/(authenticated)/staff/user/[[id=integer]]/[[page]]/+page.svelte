<script lang="ts">
	import { Tabs, TabItem } from 'flowbite-svelte';
	import {
		RectangleListSolid,
		AnnotationSolid,
		UserCircleSolid,
		LockSolid,
		TerminalSolid,
		BellSolid,
		UserEditSolid
	} from 'flowbite-svelte-icons';

	import type { PageData, Snapshot } from './$types';
	import UserProfileTab from './UserProfileTab.svelte';
	import ServiceProfileTab from './ServiceProfileTab.svelte';
	import TrainingTab from './TrainingTab.svelte';
	import HoldTab from './HoldTab.svelte';
	import NotificationTab from './NotificationTab.svelte';
	import UserUpdateTab from './UserUpdateTab.svelte';
	import ApikeyTab from './ApikeyTab.svelte';
	import AdminTab from './AdminTab.svelte';

	export let data: PageData;
	const { tabs, user } = data;
	let { tabsOpen, target } = data;

	let trainingShowDeleted = false;
	let holdsShowDeleted = false;
	let notificationsShowDeleted = false;
	let apikeysShowDeleted = false;

	type Data = {
		tabsOpen: typeof tabsOpen;
		trainingShowDeleted: boolean;
		holdsShowDeleted: boolean;
		notificationsShowDeleted: boolean;
		apikeysShowDeleted: boolean;
	};

	export const snapshot: Snapshot<Data> = {
		capture: () => {
			return {
				tabsOpen: tabsOpen,
				trainingShowDeleted: trainingShowDeleted,
				holdsShowDeleted: holdsShowDeleted,
				notificationsShowDeleted: notificationsShowDeleted,
				apikeysShowDeleted: apikeysShowDeleted
			};
		},
		restore: (value) => {
			tabsOpen = value.tabsOpen;
			trainingShowDeleted = value.trainingShowDeleted;
			holdsShowDeleted = value.holdsShowDeleted;
			notificationsShowDeleted = value.notificationsShowDeleted;
			apikeysShowDeleted = value.apikeysShowDeleted;
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

			<TrainingTab {target} bind:showDeleted={trainingShowDeleted} />
		</TabItem>
	{/if}
	{#if tabs.holds}
		<TabItem bind:open={tabsOpen.holds}>
			<div slot="title" class="flex items-center gap-2">
				<AnnotationSolid size="sm" />
				Holds
			</div>

			<HoldTab {target} bind:showDeleted={holdsShowDeleted} />
		</TabItem>
	{/if}
	{#if tabs.notifications}
		<TabItem bind:open={tabsOpen.notifications}>
			<div slot="title" class="flex items-center gap-2">
				<BellSolid size="sm" />
				Nofications
			</div>

			<NotificationTab {target} bind:showDeleted={notificationsShowDeleted} />
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

			<ApikeyTab {target} bind:showDeleted={apikeysShowDeleted} />
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
