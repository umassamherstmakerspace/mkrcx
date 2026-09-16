<script lang="ts">
	import { fleetLabels, printerStates } from './printer-state';
	export let lifecycle: string | undefined = undefined;
	$: placement = printerStates({ lifecycle }).lifecycle;
</script>

{#if placement !== 'active'}<span class="stamp">{fleetLabels[placement]}</span>{/if}

<style>
	.stamp {
		display: inline-block;
		border: 1px dashed currentColor;
		border-radius: 2px;
		padding: 0.12rem 0.35rem;
		font-size: 0.65rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: #66707c;
		white-space: nowrap;
		vertical-align: middle;
	}
	:global(.dark) .stamp {
		color: #aab3c0;
	}
</style>
