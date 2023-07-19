<script lang="ts">
  import ThemeSwitcher from "./ThemeSwitcher.svelte";
  import Branding from "./Branding.svelte";
  import { Button, Group } from "@svelteuidev/core";

  export let menu: boolean;
</script>

<Group override={{ height: '100%', px: 20 }} position="apart">
    <Branding />
    <ThemeSwitcher />
    <Button
        size="sm"
        on:click={() => {
            menu = !menu;
        }}
        ></Button>
</Group>
