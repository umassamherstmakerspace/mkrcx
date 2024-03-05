<script lang="ts">
	import { User } from '$lib/leash';
	import {
		CloseButton,
		Drawer,
		Sidebar,
		SidebarDropdownItem,
		SidebarDropdownWrapper,
		SidebarGroup,
		SidebarItem,
		SidebarWrapper
	} from 'flowbite-svelte';
	import { DrawSquareSolid, HomeSolid, LockSolid, UploadSolid } from 'flowbite-svelte-icons';
	import { sineIn } from 'svelte/easing';

	export let hidden: boolean;
	export let user: User | null;

	let transitionParams = {
		x: -320,
		duration: 200,
		easing: sineIn
	};

	const iconClass =
		'h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white';
</script>

<Drawer transitionType="fly" {transitionParams} bind:hidden id="sidebar2">
	<div class="flex items-center">
		<h5
			id="drawer-navigation-label-3"
			class="text-base font-semibold uppercase text-gray-500 dark:text-gray-400"
		>
			Menu
		</h5>
		<CloseButton on:click={() => (hidden = true)} class="mb-4 dark:text-white" />
	</div>
	<Sidebar>
		<SidebarWrapper divClass="overflow-y-auto py-4 px-3 rounded dark:bg-gray-800">
			<SidebarGroup>
				<SidebarItem label="Home" href="/">
					<svelte:fragment slot="icon">
						<HomeSolid class={iconClass} />
					</svelte:fragment>
				</SidebarItem>
				{#if user}
					<!-- <SidebarItem label="File Upload" href="/wormhole">
						<svelte:fragment slot="icon">
							<UploadSolid class={iconClass} />
						</svelte:fragment>
					</SidebarItem>
					<SidebarDropdownWrapper label="3D Printing">
						<svelte:fragment slot="icon">
							<DrawSquareSolid class={iconClass} />
						</svelte:fragment>
						<SidebarDropdownItem label="Home" href="/spectrum" />
						<SidebarDropdownItem label="Printers" href="/spectrum/printers" />
						<SidebarDropdownItem label="My Prints" href="/spectrum/my-prints" />
					</SidebarDropdownWrapper> -->
					{#if user.isStaff}
						<SidebarDropdownWrapper label="Staff Zone">
							<svelte:fragment slot="icon">
								<LockSolid class={iconClass} />
							</svelte:fragment>
							<SidebarDropdownItem label="Home" href="/staff" />
							<SidebarDropdownItem label="User Directory" href="/staff/directory" />
							<SidebarDropdownItem label="Makerspace Wifi Portal" href="/staff/wifi" />
						</SidebarDropdownWrapper>
					{/if}
				{/if}
			</SidebarGroup>
		</SidebarWrapper>
	</Sidebar>
</Drawer>
