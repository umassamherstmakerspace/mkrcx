<script lang="ts">
	import {
		Tabs,
		TabItem,
		Label,
		Input,
		Select,
		NumberInput,
		Button,
		Helper,
		Table,
		TableHead,
		TableHeadCell,
		TableBody,
		TableBodyRow,
		TableBodyCell,
		Alert,
		Badge,
		Indicator
	} from 'flowbite-svelte';
	import { UserCircleSolid, RectangleListSolid, AnnotationSolid } from 'flowbite-svelte-icons';
	import UserCell from '$lib/components/UserCell.svelte';

	import type { UserUpdateOptions } from '$lib/leash';
	import type { ConvertFields } from '$lib/types';
	import type { PageData, Snapshot } from './$types';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import { page } from '$app/stores';

	export let data: PageData;
	let { target } = data;

	let profileError: string = '';

	let tabsOpen = {
		profile: true,
		trainings: false,
		holds: false,
		apikeys: false,
		updates: false
	};

	type Data = {
		tabsOpen: typeof tabsOpen;
	};

	function setTab(tab: keyof typeof tabsOpen) {
		Object.keys(tabsOpen).forEach((key) => {
			tabsOpen[key as keyof typeof tabsOpen] = false;
		});
		tabsOpen[tab] = true;

		tabsOpen = { ...tabsOpen };
	}

	$: switch ($page.url.hash) {
		case '#trainings':
			setTab('trainings');
			break;
		case '#holds':
			setTab('holds');
			break;
		case '#updates':
			setTab('updates');
			break;
		default:
			setTab('profile');
			break;
	}

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

	let changed = false;
	const change = () => (changed = true);

	const userUpdate: UserUpdateOptions = {
		name: '',
		email: '',
		type: '',
		major: undefined,
		graduationYear: undefined
	};

	const userUpdateError: ConvertFields<UserUpdateOptions, string> = {};

	function loadUser() {
		userUpdate.name = target.name;
		userUpdate.email = target.email;
		if (target.pendingEmail) {
			userUpdate.email = target.pendingEmail;
		}
		userUpdate.type = target.type;
		if (target.type == 'undergrad' || target.type == 'grad' || target.type == 'alumni') {
			userUpdate.major = target.major;
			userUpdate.graduationYear = target.graduationYear;
		}

		changed = false;

		Object.keys(userUpdateError).forEach((key) => {
			userUpdateError[key as keyof UserUpdateOptions] = undefined;
		});
	}

	loadUser();

	const studentLike =
		target.type == 'undergrad' || target.type == 'grad' || target.type == 'alumni';

	const isError = (value: string | undefined) => {
		return value != undefined;
	};

	const labelColor = (value: string | undefined) => {
		if (isError(value)) return 'red';
		return 'gray';
	};

	const inputColor = (value: string | undefined) => {
		if (isError(value)) return 'red';
		return 'base';
	};

	function validate(): boolean {
		const emailRe =
			/^(([^<>()[\].,;:\s@"]+(\.[^<>()[\].,;:\s@"]+)*)|(".+"))@(([^<>()[\].,;:\s@"]+\.)+[^<>()[\].,;:\s@"]{2,})$/i;
		let hasError = false;
		if (!userUpdate.name) {
			userUpdateError.name = 'Name cannot be empty';
			hasError = true;
		} else {
			userUpdateError.name = undefined;
		}

		if (!userUpdate.email) {
			userUpdateError.email = 'Email cannot be empty';
			hasError = true;
		} else if (!emailRe.test(userUpdate.email)) {
			userUpdateError.email = 'Email is invalid';
			hasError = true;
		} else {
			userUpdateError.email = undefined;
		}

		if (studentLike) {
			if (!userUpdate.major) {
				userUpdateError.major = 'Major cannot be empty';
				hasError = true;
			} else {
				userUpdateError.major = undefined;
			}

			if (!userUpdate.graduationYear) {
				userUpdateError.graduationYear = 'Invalid graduation year';
				hasError = true;
			} else {
				userUpdateError.graduationYear = undefined;
			}
		}

		return hasError;
	}

	async function updateUser() {
		if (validate()) return;

		try {
			target = await target.update(userUpdate);
			loadUser();
		} catch (error) {
			console.error(error);
			let message = '';
			if (error instanceof Error) {
				message = error.message;
			} else {
				message = String(error);
			}

			profileError = message;
		}
	}

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
		{#if profileError}
			<Alert border color="red" dismissable on:close={() => (profileError = '')}>
				<span class="font-medium">Error: </span>
				{profileError}
			</Alert>
		{/if}
		<form on:submit|preventDefault={updateUser}>
			<div class="flex flex-col space-y-10">
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.name)} for="name-input" class="mb-2 block"
						>Name</Label
					>
					<Input
						color={inputColor(userUpdateError.name)}
						bind:value={userUpdate.name}
						on:input={change}
						on:change={validate}
						type="text"
						id="name-input"
					/>
					{#if isError(userUpdateError.name)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.name}
						</Helper>
					{/if}
				</div>
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.email)} for="email-input" class="mb-2 block"
						>Email</Label
					>
					<Input
						color={inputColor(userUpdateError.email)}
						bind:value={userUpdate.email}
						on:input={change}
						on:change={validate}
						type="email"
						id="email-input"
					/>
					{#if isError(userUpdateError.email)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.email}
						</Helper>
					{/if}
					{#if target.pendingEmail}
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Pending Email: {target.pendingEmail}
						</p>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Current Email: {target.email}
						</p>
					{/if}
				</div>
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.type)} for="type-selector" class="mb-2 block"
						>User Type</Label
					>
					<Select
						color={inputColor(userUpdateError.type)}
						bind:value={userUpdate.type}
						on:input={change}
						on:change={validate}
						id="type-selector"
					>
						<option value="undergrad">Undergraduate Student</option>
						<option value="grad">Graduate Student</option>
						<option value="alumni">Alumni</option>
						<option value="faculty">Faculty</option>
						<option value="staff">Staff</option>
						<option value="other">Other</option>
					</Select>
					{#if isError(userUpdateError.type)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.type}
						</Helper>
					{/if}
				</div>
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.major)} for="major-input" class="mb-2 block"
						>Major</Label
					>
					<Input
						color={inputColor(userUpdateError.major)}
						bind:value={userUpdate.major}
						on:input={change}
						on:change={validate}
						type="text"
						id="major-input"
					/>
					{#if isError(userUpdateError.major)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.major}
						</Helper>
					{/if}
				</div>
				<div class="flex flex-col justify-between">
					<Label
						color={labelColor(userUpdateError.graduationYear)}
						for="graduation-year-input"
						class="mb-2 block">Graduation Year</Label
					>
					<NumberInput
						color={inputColor(userUpdateError.graduationYear)}
						bind:value={userUpdate.graduationYear}
						on:input={change}
						on:change={validate}
						type="number"
						id="graduation-year-input"
					/>
					{#if isError(userUpdateError.graduationYear)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.graduationYear}
						</Helper>
					{/if}
				</div>
				<div class="flex justify-end">
					<Button color="yellow" disabled={!changed} class="w-1/4" type="submit">Save</Button>
				</div>
			</div>
		</form>
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
