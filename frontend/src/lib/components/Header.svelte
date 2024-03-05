<script lang="ts">
	import {
		Alert,
		Avatar,
		Button,
		CloseButton,
		Dropdown,
		DropdownDivider,
		DropdownHeader,
		DropdownItem,
		NavBrand,
		NavHamburger,
		Navbar,
		ToolbarButton
	} from 'flowbite-svelte';
	import { BellSolid, DotsVerticalOutline, EyeSolid } from 'flowbite-svelte-icons';
	import DropdownTheme from './DropdownTheme.svelte';
	import { Hold, Notification, User } from '$lib/leash';
	import Timestamp from './Timestamp.svelte';

	export let hideSidebar: boolean;
	export let user: User | null;

	function holdSort(a: Hold, b: Hold) {
		if (a.priority < b.priority) return -1;
		if (a.priority > b.priority) return 1;
		if (a.createdAt < b.createdAt) return -1;
		if (a.createdAt > b.createdAt) return 1;
		return 0;
	}

	let notifications: Promise<Notification[]> = user?.getAllNotifications() ?? Promise.resolve([]);
	let holds: Promise<Hold[]> =
		user?.getAllHolds().then((holds) => holds.filter((h) => h.isActive).sort(holdSort)) ??
		Promise.resolve([]);

	async function clearNotification(notification: Notification) {
		await notification.delete();

		if (!user) return;
		user = await user.get({ withNotifications: true, withHolds: true });
		notifications = user.getAllNotifications();
	}

	async function clearNotifications() {
		if (!user) return;

		(await notifications).forEach((notification) => notification.delete());
		user = await user.get({ withNotifications: true, withHolds: true });
		notifications = user.getAllNotifications();
	}

	function holdColor(hold: Hold) {
		if (hold.priority < 100) return 'red';
		return 'yellow';
	}
</script>

<Navbar>
	<div class="flex flex-1 items-center space-x-6 md:order-1">
		<NavHamburger onClick={() => (hideSidebar = false)} class="m-0 ml-3 sm:hidden md:block" />
		<NavBrand href="/">
			<span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white"
				>UMass Makerspace</span
			>
		</NavBrand>
	</div>
	<div class="flex items-center space-x-3 md:order-2">
		{#if user}
			<div
				id="bell"
				class="inline-flex items-center text-center text-sm font-medium text-gray-500 hover:text-gray-900 focus:outline-none dark:text-gray-400 dark:hover:text-white"
			>
				<BellSolid class="h-5kj w-5" />
				{#await notifications then notifications}
					{#if notifications.length > 0}
						<div class="relative flex">
							<div
								class="relative -top-2 end-3 inline-flex h-3 w-3 rounded-full border-2 border-white bg-red-500 dark:border-gray-900"
							/>
						</div>
					{/if}
				{/await}
			</div>
			<Avatar id="avatar-menu" src={user.iconURL} rounded />
		{:else}
			<ToolbarButton>
				<DotsVerticalOutline id="dots-menu" class="dark:text-white" />
			</ToolbarButton>
			<Button size="sm" href="/login" variant="primary">Login</Button>
		{/if}
	</div>
	{#if user}
		<Dropdown placement="bottom" triggeredBy="#avatar-menu" class="space-y-3 p-3 text-sm">
			<DropdownHeader>
				<span class="block text-sm">{user.name}</span>
				<span class="block truncate text-sm font-medium">{user.email}</span>
			</DropdownHeader>
			<DropdownItem href="/profile">Profile</DropdownItem>
			<DropdownItem href="/profile/checkin">Check In</DropdownItem>
			<DropdownDivider />
			<DropdownItem href="/settings">Settings</DropdownItem>
			<DropdownTheme />
			<DropdownDivider />
			<DropdownItem href="/logout">Logout</DropdownItem>
		</Dropdown>
		<Dropdown
			triggeredBy="#bell"
			class="max-h-72 w-full min-w-64 max-w-md divide-y divide-gray-100 overflow-y-auto rounded shadow dark:divide-gray-700 dark:bg-gray-800"
		>
			<div slot="header" class="py-2 text-center font-bold">Notifications</div>
			{#await notifications then notifications}
				{#if notifications.length > 0}
					{#each notifications as notification}
						<DropdownItem class="flex space-x-4 rtl:space-x-reverse">
							<div class="flex w-full gap-3 ps-3">
								<div class="flex-shrink-0">
									<div class="font-bold">{notification.title}</div>
									<div class="mb-1.5 text-sm text-gray-500 dark:text-gray-400">
										{notification.message}
									</div>
									<div class="xtext-primary-600 text-xs dark:text-primary-500">
										<Timestamp timestamp={notification.createdAt} formatter="relative" />
									</div>
								</div>
								<CloseButton on:click={() => clearNotification(notification)} />
							</div>
						</DropdownItem>
					{/each}
					<DropdownItem on:click={clearNotifications}>
						<div class="inline-flex items-center">
							<EyeSolid class="me-2 h-4 w-4 text-gray-500 dark:text-gray-400" />
							Clear all
						</div>
					</DropdownItem>
				{:else}
					<div class="px-4 py-2 text-center text-sm font-medium">No notifications</div>
				{/if}
			{/await}
		</Dropdown>
	{:else}
		<Dropdown placement="bottom" triggeredBy="#dots-menu" class="space-y-3 p-3 text-sm">
			<DropdownTheme />
			<DropdownItem href="/settings">Settings</DropdownItem>
			<DropdownItem />
		</Dropdown>
	{/if}
</Navbar>
{#if user}
	{#await holds then holds}
		{#if holds.length > 0}
			<Alert border class="text-center" color={holdColor(holds[0])}>
				{holds[0].reason}
			</Alert>
		{:else if user.pendingEmail}
			<Alert border class="text-center" color="yellow">
				Your email is pending verification. Please log in with your new email.
			</Alert>
		{/if}
	{/await}
{/if}
