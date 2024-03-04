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
		Indicator,

		Button,

		CloseButton


	} from 'flowbite-svelte';
	import { RectangleListSolid, AnnotationSolid, UserCircleSolid } from 'flowbite-svelte-icons';
	import UserCell from '$lib/components/UserCell.svelte';

	import type { PageData, Snapshot } from './$types';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserProfileTab from './UserProfileTab.svelte';
	import ServiceProfileTab from './ServiceProfileTab.svelte';
	import CreateTrainingModal from '$lib/components/modals/CreateTrainingModal.svelte';
	import { timeout, type ModalOptions } from '$lib/components/modals/modals';
	import type { DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import type { Hold, Training } from '$lib/leash';
	import CreateHoldModal from '$lib/components/modals/CreateHoldModal.svelte';
	import TrainingTab from '../TrainingTab.svelte';
	import HoldTab from '../HoldTab.svelte';

	export let data: PageData;
	const { user, tabs } = data;
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

		<TrainingTab {target} />
	</TabItem>
	<TabItem bind:open={tabsOpen.holds}>
		<div slot="title" class="flex items-center gap-2">
			<AnnotationSolid size="sm" />
			Holds
		</div>

		<HoldTab {target} />
	</TabItem>
</Tabs>
