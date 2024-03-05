<script lang="ts">
	import Timestamp from '$lib/components/Timestamp.svelte';
	import UserCell from '$lib/components/UserCell.svelte';
	import CreateHoldModal from '$lib/components/modals/CreateHoldModal.svelte';
	import DeleteModal, { type DeleteModalOptions } from '$lib/components/modals/DeleteModal.svelte';
	import { timeout, type ModalOptions } from '$lib/components/modals/modals';
	import type { Hold, User } from '$lib/leash';
	import {
		Badge,
		Button,
		CloseButton,
		Indicator,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell
	} from 'flowbite-svelte';

	export let target: User;

</script>
TODO: Implement the rest of the file