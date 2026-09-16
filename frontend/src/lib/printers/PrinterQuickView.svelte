<script lang="ts">
	import type { Printer } from './prototype-data';
	import { fleetLabels, maintenanceLabels, printerStates } from './printer-state';
	export let printer: Printer;
	$: states = printerStates(printer);
</script>

<div class="quick-view">
	<dl>
		<div>
			<dt class="sr-only">Fleet</dt>
			<dd>{fleetLabels[states.lifecycle]}</dd>
		</div>
		{#if states.maintenance !== 'none'}<div>
				<dt>Maintenance</dt>
				<dd>{maintenanceLabels[states.maintenance]}</dd>
			</div>{/if}
		{#if printer.location}<div>
				<dt>Location</dt>
				<dd>{printer.location}</dd>
			</div>{/if}
		{#if printer.machineId}<div>
				<dt>Machine</dt>
				<dd>{printer.machineId}</dd>
			</div>{/if}
	</dl>
	{#if !printer.stale && printer.job && ['printing', 'paused'].includes(printer.activity)}
		<p class="file">{printer.job.file}</p>
		<p>
			{[
				printer.job.person,
				printer.job.material,
				printer.progress === undefined ? '' : `${Math.round(printer.progress)}%`
			]
				.filter(Boolean)
				.join(' · ')}
		</p>
	{/if}
</div>

<style>
	.quick-view {
		font-size: 0.85rem;
		padding: 0.5rem 0;
		font-weight: 400;
	}
	dl {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem 1.5rem;
	}
	dl div {
		display: flex;
		gap: 0.4rem;
	}
	dt {
		color: #66707c;
	}
	p {
		overflow-wrap: anywhere;
	}
	.file {
		margin-top: 0.35rem;
	}
	:global(.dark) dt {
		color: #aab3c0;
	}
</style>
