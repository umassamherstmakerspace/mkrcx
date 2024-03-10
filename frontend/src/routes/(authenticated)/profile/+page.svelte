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
		Alert
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

	let profileError: string = '';

	let tabsOpen = {
		profile: true,
		trainings: false,
		holds: false
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
			user = await user.update(userUpdate);
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
		const holds = await user.getAllHolds();
		return holds.filter((hold) => {
			if (hold.end == undefined) return true;
			return hold.end.getTime() > Date.now();
		});
	}
</script>

<svelte:head>
	<title>mkr.cx | Profile</title>
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
				<TableHeadCell>Name</TableHeadCell>
				<TableHeadCell>Level</TableHeadCell>
				<TableHeadCell>Date Added</TableHeadCell>
			</TableHead>
			<TableBody>
				{#await user.getAllTrainings()}
					<TableBodyRow>
						<TableBodyCell colspan="3" class="p-0">Loading...</TableBodyCell>
					</TableBodyRow>
				{:then trainings}
					{#each trainings as training}
						<TableBodyRow>
							<TableBodyCell>{training.name}</TableBodyCell>
							<TableBodyCell>{training.levelString()}</TableBodyCell>
							<TableBodyCell><Timestamp timestamp={training.createdAt} /></TableBodyCell>
						</TableBodyRow>
					{/each}
				{:catch error}
					<TableBodyRow>
						<TableBodyCell colspan="3" class="p-0">Error: {error.message}</TableBodyCell>
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
				<TableHeadCell>Resolve</TableHeadCell>
			</TableHead>
			<TableBody>
				{#await getHolds()}
					<TableBodyRow>
						<TableBodyCell colspan="5" class="p-0">Loading...</TableBodyCell>
					</TableBodyRow>
				{:then holds}
					{#each holds as hold}
						<TableBodyRow>
							<TableBodyCell>{hold.name}</TableBodyCell>
							<TableBodyCell>{hold.reason}</TableBodyCell>
							<TableBodyCell>
								{#if hold.start}
									<Timestamp timestamp={hold.start} />
								{:else}
									-
								{/if}
							</TableBodyCell>
							<TableBodyCell>
								{#if hold.end}
									<Timestamp timestamp={hold.end} />
								{:else}
									-
								{/if}
							</TableBodyCell>
							<TableBodyCell>
								{#if hold.resolutionLink}
									<a href={hold.resolutionLink} target="_blank" rel="noopener noreferrer">Resolve</a
									>
								{:else}
									-
								{/if}
							</TableBodyCell>
						</TableBodyRow>
					{/each}
				{:catch error}
					<TableBodyRow>
						<TableBodyCell colspan="5" class="p-0">Error: {error.message}</TableBodyCell>
					</TableBodyRow>
				{/await}
			</TableBody>
		</Table>
	</TabItem>
</Tabs>
