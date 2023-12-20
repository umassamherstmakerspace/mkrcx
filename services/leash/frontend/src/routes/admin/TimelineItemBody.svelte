<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import type { UserTimelineItem } from './userCard';
	import { Text } from '@svelteuidev/core';

	export let timelineItem: UserTimelineItem;
</script>

{#if timelineItem.element.elementType === 'user'}
	<Text size="xs"><Timestamp time={timelineItem.timestamp} /></Text>
{:else if timelineItem.element.elementType === 'training'}
	<Text color="dimmed" size="sm">
		{timelineItem.element.trainingItem.action === 'created' ? 'Added' : 'Removed'} training
		<Text variant="link" root="span" href="#" inherit>
			{timelineItem.element.trainingItem.trainingType}
		</Text>
		by
		<Text variant="link" root="span" href="#" inherit>
			{timelineItem.element.trainingItem.action === 'created'
				? timelineItem.element.trainingItem.addedBy.name
				: timelineItem.element.trainingItem.removedBy?.name}
		</Text>
	</Text>
	<Text size="xs"><Timestamp time={timelineItem.timestamp} /></Text>
{:else if timelineItem.element.elementType === 'userUpdate'}
	<Text color="dimmed" size="sm">
		<Text variant="link" root="span" href="#" inherit>
			{timelineItem.element.userUpdate.field}
		</Text>
		{#if !timelineItem.element.userUpdate.oldValue}
			set to
			<Text variant="link" root="span" href="#" inherit>
				{timelineItem.element.userUpdate.newValue}
			</Text>
		{:else}
			updated from
			<Text variant="link" root="span" href="#" inherit>
				{timelineItem.element.userUpdate.oldValue}
			</Text>
			to
			<Text variant="link" root="span" href="#" inherit>
				{timelineItem.element.userUpdate.newValue}
			</Text>
		{/if}
		by
		<Text variant="link" root="span" href="#" inherit>
			{timelineItem.element.userUpdate.editedBy.name}
		</Text>
	</Text>
	<Text size="xs"><Timestamp time={timelineItem.timestamp} /></Text>
{/if}
