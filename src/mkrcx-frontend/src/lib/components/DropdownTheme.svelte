<script lang="ts">
	import { theme } from "$lib/stores";
	import { ButtonGroup, Label, RadioButton } from "flowbite-svelte";
</script>

<li class="py-2 px-4">
    <Label for="theme-selector-dropdown" class="mb-2 block">Theme</Label>
    <ButtonGroup id="theme-selector-dropdown">
        <RadioButton value={'dark'} bind:group={$theme}>Dark</RadioButton>
        <RadioButton value={'light'} bind:group={$theme}>Light</RadioButton>
        <RadioButton value={'system'} bind:group={$theme}>System</RadioButton>
    </ButtonGroup>
</li>