<script lang="ts">
  import ThemeSwitcher from "./ThemeSwitcher.svelte";
  import Branding from "./Branding.svelte";
  import { Group } from "@svelteuidev/core";
</script>

<Group override={{ height: '100%', px: 20 }} position="apart">
    <Branding />
    <ThemeSwitcher />
</Group>
