<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import type { User } from '$lib/leash';
	import {
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';

	export let target: User;
</script>

<Table>
	<TableHead>
		<TableHeadCell>Field</TableHeadCell>
		<TableHeadCell>Old Value</TableHeadCell>
		<TableHeadCell>New Value</TableHeadCell>
		<TableHeadCell>Updated At</TableHeadCell>
		<TableHeadCell>Updated By</TableHeadCell>
	</TableHead>
	<TableBody>
		{#await target.getAllUserUpdates(false, true)}
			<TableBodyRow>
				<TableBodyCell colspan="5" class="p-0">Loading...</TableBodyCell>
			</TableBodyRow>
		{:then updates}
			{#each updates as update}
				<TableBodyRow>
					<TableBodyCell>{update.field}</TableBodyCell>
					<TableBodyCell>{update.oldValue}</TableBodyCell>
					<TableBodyCell>{update.newValue}</TableBodyCell>
					<TableBodyCell><Timestamp timestamp={update.createdAt} /></TableBodyCell>
					<TableBodyCell><UserCell user={update.getEditedBy()} /></TableBodyCell>
				</TableBodyRow>
			{/each}
		{:catch error}
			<TableBodyRow>
				<TableBodyCell colspan="5" class="p-0">Error: {error.message}</TableBodyCell>
			</TableBodyRow>
		{/await}
	</TableBody>
</Table>
