<script lang="ts">
	import { Avatar, Button, Dropdown, DropdownDivider, DropdownHeader, DropdownItem, NavBrand, NavHamburger, Navbar, ToolbarButton } from "flowbite-svelte";
	import { DotsVerticalOutline } from "flowbite-svelte-icons";
    import { user } from "$lib/stores";
    import DropdownTheme from "./DropdownTheme.svelte";
	import { logout } from "$lib/leash";
	import { page } from "$app/stores";

    export let hideSidebar: boolean;
</script>

<Navbar>
	<div class="flex flex-1 items-center space-x-6 md:order-1">
	<NavHamburger onClick={() => hideSidebar = false} class="ml-3 m-0 sm:hidden md:block" />
	<NavBrand href="/">
		<span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white"
			>UMass Makerspace</span
		>
	</NavBrand>
	</div>
	<div class="flex items-center space-x-3 md:order-2">
		{#if $user}
			<Avatar id="avatar-menu" src={$user.iconURL} rounded />
		{:else}
		<ToolbarButton>
			<DotsVerticalOutline id="dots-menu" class="dark:text-white" />
		</ToolbarButton>
			<Button size="sm" href="/login" variant="primary">Login</Button>
		{/if}
	</div>
	<Dropdown placement="bottom" triggeredBy="#dots-menu" class="space-y-3 p-3 text-sm">
        <DropdownTheme />
		<DropdownItem href="/settings">Settings</DropdownItem>
		<DropdownItem />
	</Dropdown>
	{#if $user}
		<Dropdown placement="bottom" triggeredBy="#avatar-menu" class="space-y-3 p-3 text-sm">
			<DropdownHeader>
				<span class="block text-sm">{$user.name}</span>
				<span class="block truncate text-sm font-medium">{$user.email}</span>
			</DropdownHeader>
			<DropdownItem href="/profile">Profile</DropdownItem>
			<DropdownDivider />
			<DropdownItem href="/settings">Settings</DropdownItem>
			<DropdownTheme />
			<DropdownDivider />
			<DropdownItem on:click={() => logout($page.url.origin)}>Logout</DropdownItem>
		</Dropdown>
	{/if}
</Navbar>