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
		TableBodyCell
	} from 'flowbite-svelte';
	import { UserCircleSolid, RectangleListSolid, AnnotationSolid } from 'flowbite-svelte-icons';

	import type { UserUpdateOptions } from '$lib/leash';
	import type { ConvertFields } from '$lib/types';
	import type { PageData } from './$types';
	import Timestamp from '$lib/components/Timestamp.svelte';
	import { page } from '$app/stores';
	import type { Snapshot } from './$types';

	export let data: PageData;
	let { user } = data;

	let tabsOpen = {
		profile: true,
		trainings: false,
		holds: false,
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
		userUpdate.name = user.name;
		userUpdate.email = user.email;
		if (user.pendingEmail) {
			userUpdate.email = user.pendingEmail;
		}
		userUpdate.type = user.type;
		if (user.type == 'undergrad' || user.type == 'grad' || user.type == 'alumni') {
			userUpdate.major = user.major;
			userUpdate.graduationYear = user.graduationYear;
		}

		changed = false;

		Object.keys(userUpdateError).forEach((key) => {
			userUpdateError[key as keyof UserUpdateOptions] = undefined;
		});
	}

	loadUser();

	const studentLike = user.type == 'undergrad' || user.type == 'grad' || user.type == 'alumni';

	const isError = (value: any | undefined) => {
		return value != undefined;
	};

	const labelColor = (value: any | undefined) => {
		if (isError(value)) return 'red';
		return 'gray';
	};

	const inputColor = (value: any | undefined) => {
		if (isError(value)) return 'red';
		return 'base';
	};

	function validate(): boolean {
		const emailRe =
			/^(([^<>()[\]\.,;:\s@\"]+(\.[^<>()[\]\.,;:\s@\"]+)*)|(\".+\"))@(([^<>()[\]\.,;:\s@\"]+\.)+[^<>()[\]\.,;:\s@\"]{2,})$/i;
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
			user = await user.update(userUpdate);
			loadUser();
		} catch (error) {
			console.error(error);
		}
	}

	async function getHolds() {
		const holds = await user.getAllHolds();
		return holds.filter((hold) => {
			if (hold.holdEnd == undefined) return true;
			return hold.holdEnd.getTime() > Date.now();
		});
	}
</script>

<Tabs style="underline">
	<TabItem bind:open={tabsOpen.profile}>
		<div slot="title" class="flex items-center gap-2">
			<UserCircleSolid size="sm" />
			Profile
		</div>
		<form on:submit|preventDefault={updateUser}>
			<div class="flex flex-col space-y-10">
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.name)} for="large-input" class="mb-2 block"
						>Name</Label
					>
					<Input
						color={inputColor(userUpdateError.name)}
						bind:value={userUpdate.name}
						on:input={change}
						on:change={validate}
						type="text"
					/>
					{#if isError(userUpdateError.name)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.name}
						</Helper>
					{/if}
				</div>
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.email)} for="large-input" class="mb-2 block"
						>Email</Label
					>
					<Input
						color={inputColor(userUpdateError.email)}
						bind:value={userUpdate.email}
						on:input={change}
						on:change={validate}
						type="email"
					/>
					{#if isError(userUpdateError.email)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.email}
						</Helper>
					{/if}
					{#if user.pendingEmail}
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Your email is pending verification. Set it back to <b>{user.email}</b> to revert.
						</p>
					{/if}
					<p class="text-sm text-gray-500 dark:text-gray-400">
						<b>Warning:</b>
						Your email will change once when you log out and log back in with the new email.
					</p>
				</div>
				<div class="flex flex-col justify-between">
					<Label color={labelColor(userUpdateError.type)} for="large-input" class="mb-2 block"
						>User Type</Label
					>
					<Select
						color={inputColor(userUpdateError.type)}
						bind:value={userUpdate.type}
						on:input={change}
						on:change={validate}
					>
						{#if studentLike}
							<option value="undergrad">Undergraduate Student</option>
							<option value="grad">Graduate Student</option>
							<option value="alumni">Alumni</option>
						{:else if userUpdate.type == 'faculty'}
							<option value="faculty">Faculty</option>
						{:else if userUpdate.type == 'staff'}
							<option value="staff">Staff</option>
						{:else if userUpdate.type == 'other'}
							<option value="other">Other</option>
						{/if}
					</Select>
					{#if isError(userUpdateError.type)}
						<Helper class="mt-2" color="red">
							<span class="font-medium">Error:</span>
							{userUpdateError.type}
						</Helper>
					{/if}
				</div>
				{#if studentLike}
					<div class="flex flex-col justify-between">
						<Label color={labelColor(userUpdateError.major)} for="large-input" class="mb-2 block"
							>Major</Label
						>
						<Input
							color={inputColor(userUpdateError.major)}
							bind:value={userUpdate.major}
							on:input={change}
							on:change={validate}
							type="text"
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
							for="large-input"
							class="mb-2 block">Graduation Year</Label
						>
						<NumberInput
							color={inputColor(userUpdateError.graduationYear)}
							bind:value={userUpdate.graduationYear}
							on:input={change}
							on:change={validate}
							type="number"
						/>
						{#if isError(userUpdateError.graduationYear)}
							<Helper class="mt-2" color="red">
								<span class="font-medium">Error:</span>
								{userUpdateError.graduationYear}
							</Helper>
						{/if}
					</div>
				{/if}
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
				<TableHeadCell>Training Type</TableHeadCell>
				<TableHeadCell>Date Added</TableHeadCell>
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
							<TableBodyCell><Timestamp timestamp={training.createdAt} /></TableBodyCell>
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
				<TableHeadCell>Hold Type</TableHeadCell>
				<TableHeadCell>Reason</TableHeadCell>
				<TableHeadCell>Start Date</TableHeadCell>
				<TableHeadCell>End Date</TableHeadCell>
			</TableHead>
			<TableBody>
				{#await getHolds()}
					<TableBodyRow>
						<TableBodyCell colspan="2" class="p-0">Loading...</TableBodyCell>
					</TableBodyRow>
				{:then holds}
					{#each holds as hold}
						<TableBodyRow>
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
