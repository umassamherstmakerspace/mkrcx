<script lang="ts">
	import UserRow from "$lib/components/UserRow.svelte";
	import { User } from "$lib/leash";
	import { P, Skeleton, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, TableSearch } from "flowbite-svelte";
    import { inview, type Options as InviewOptions } from 'svelte-inview';

    let query = "";

    let users: User[] = [];
    let offset = 0;
    let hasMore = true;
    let loaded = false;

    const inviewOptions: InviewOptions = {
    rootMargin: '50px',
  };

    async function search(q: string) {
        if (!hasMore) return;
        loaded = false;

        const res = await User.search(q, {
            offset,
            limit: 50,
            withHolds: true,
            withTrainings: true,
        });

        users = [...users, ...res.data];
        offset += res.data.length;
        hasMore = res.total > offset;
        loaded = true;
    }

    function newSearch(q: string) {
        loaded = false;
        offset = 0;
        users = [];
        hasMore = true;
        search(q);
    }

    $: newSearch(query);
    
    let activeRow: number | null = null;

    function toggleRow(i: number) {
        activeRow = activeRow === i ? null : i;
    }
</script>

<div class="flex flex-col p-4 w-full">
    <TableSearch placeholder="Search by name or email..." hoverable={true} bind:inputValue={query} >
        <div class='relative overflow-x-auto overflow-y-auto max-h-fit'>
        <TableHead>
          <TableHeadCell>ID</TableHeadCell>
          <TableHeadCell>Name</TableHeadCell>
          <TableHeadCell>Email</TableHeadCell>
          <TableHeadCell>Role</TableHeadCell>
          <TableHeadCell>Type</TableHeadCell>
          <TableHeadCell>Major</TableHeadCell>
          <TableHeadCell>Graduation Year</TableHeadCell>
        </TableHead>
        <TableBody tableBodyClass="divide-y divide-gray-200 dark:divide-gray-700">
          {#each users as user, i}
            <UserRow user={user} open={activeRow === i} on:click={() => toggleRow(i)} />
          {/each}
          {#if loaded && hasMore}
          <TableBodyRow>
            <TableBodyCell colspan="8" class="p-0">
              <div class="px-2 py-3" use:inview={inviewOptions} on:inview_enter={() => search(query)}>
                <div class="animate-pulse flex flex-col items-center w-full">
                    <P size="sm" weight="light" class="text-gray-200 dark:text-gray-700 mb-2.5"> Loading More... </P>
                    <div class="h-2 bg-gray-200 rounded-full dark:bg-gray-700 mb-2.5 w-full" />
                    <div class="h-2 bg-gray-200 rounded-full dark:bg-gray-700 mb-2.5 w-full" />
                </div>
              </div>
            </TableBodyCell>
        </TableBodyRow>
        {/if}
        </TableBody>
    </div>
      </TableSearch>
</div>