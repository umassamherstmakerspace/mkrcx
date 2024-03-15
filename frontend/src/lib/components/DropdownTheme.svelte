<script lang="ts">
	import { userPrefersMode } from 'mode-watcher';
	import { ButtonGroup, Label, RadioButton } from 'flowbite-svelte';
</script>

<li class="px-4 py-2">
	<Label for="theme-selector-dropdown" class="mb-2 block">Theme</Label>
	<ButtonGroup id="theme-selector-dropdown">
		<RadioButton value={'dark'} bind:group={$userPrefersMode}>Dark</RadioButton>
		<RadioButton value={'light'} bind:group={$userPrefersMode}>Light</RadioButton>
		<RadioButton value={'system'} bind:group={$userPrefersMode}>System</RadioButton>
	</ButtonGroup>
</li>
