<script lang="ts">
	import type { User } from "$lib/leash";
	import { ImagePlaceholder, TableBodyCell, TableBodyRow } from "flowbite-svelte";
	import { slide } from "svelte/transition";


    export let user: User;
    export let open: boolean;
</script>


<TableBodyRow on:click>
    <TableBodyCell>{user.id}</TableBodyCell>
    <TableBodyCell>{user.name}</TableBodyCell>
    <TableBodyCell>{user.email}</TableBodyCell>
    <TableBodyCell>{user.role}</TableBodyCell>
    <TableBodyCell>{user.type}</TableBodyCell>
    <TableBodyCell>{user.major}</TableBodyCell>
    {#if user.graduationYear > 0}
        <TableBodyCell>{user.graduationYear}</TableBodyCell>
    {:else}
        <TableBodyCell></TableBodyCell>
    {/if}
</TableBodyRow>
{#if open}
        <TableBodyRow>
        <TableBodyCell colspan="4" class="p-0">
          <div class="px-2 py-3" transition:slide={{ duration: 300, axis: 'y' }}>
            <ImagePlaceholder />
          </div>
        </TableBodyCell>
      </TableBodyRow>
    {/if}
