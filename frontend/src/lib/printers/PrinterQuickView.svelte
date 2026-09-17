<script lang="ts">
	import type { Printer } from './prototype-data';
	import { printRecipient } from './printer-state';
	export let printer: Printer;
	const started = (value: string) => {
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? value || 'Unavailable' : date.toLocaleString();
	};
</script>

<div class="print-details">
	{#if printer.lifecycle === 'retired'}
		<p class="detail-empty">Retired printer. Open its name to view historical records.</p>
	{:else if printer.fault && !printer.stale}
		<p class="fault">{printer.fault}</p>
	{:else if printer.stale || printer.connected === false}
		<p class="detail-empty">Current print details are unavailable until the printer reconnects.</p>
	{:else if printer.job && ['printing', 'paused'].includes(printer.activity)}
		<dl class="detail-fields">
			<div>
				<dt>Printing for</dt>
				<dd>{printRecipient(printer.job.person) || 'Not recorded'}</dd>
			</div>
			<div>
				<dt>File</dt>
				<dd class="filename">{printer.job.file || 'Unavailable'}</dd>
			</div>
			<div>
				<dt>Material</dt>
				<dd>{printer.job.material || 'Unavailable'}</dd>
			</div>
			<div>
				<dt>Started</dt>
				<dd>{started(printer.job.started)}</dd>
			</div>
			{#if printer.progress !== undefined}
				<div>
					<dt>{printer.activity === 'paused' ? 'Paused' : 'Progress'}</dt>
					<dd class="detail-progress">
						<progress
							max="100"
							value={printer.progress}
							aria-label={`${printer.name} print progress`}
						></progress><span>{Math.round(printer.progress)}%</span>
					</dd>
				</div>
			{/if}
		</dl>
	{:else}
		<p class="detail-empty">
			{['printing', 'paused'].includes(printer.activity)
				? 'Print details unavailable.'
				: 'No print in progress.'}
		</p>
	{/if}
</div>

<style>
	.print-details {
		padding: 14px 18px;
		font-weight: 400;
	}
	.detail-fields {
		display: flex;
		flex-wrap: wrap;
		gap: 12px 28px;
	}
	.detail-fields > div {
		min-width: 0;
	}
	dt {
		color: var(--muted, #66707c);
		font-size: 10px;
		margin-bottom: 3px;
	}
	dd {
		font-size: 12px;
		overflow-wrap: anywhere;
	}
	.filename {
		font-family: ui-monospace, monospace;
		font-size: 11px;
	}
	.detail-progress {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	progress {
		width: 100px;
		height: 6px;
		accent-color: #881c1c;
	}
	.detail-empty {
		font-size: 12px;
		color: var(--muted, #66707c);
	}
	.fault {
		font-size: 12px;
		color: #a12a37;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	@media (max-width: 760px) {
		.print-details {
			padding: 12px 0 2px;
		}
		.detail-fields {
			display: grid;
			grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		}
		.detail-fields > div:nth-child(2) {
			grid-column: 1 / -1;
			grid-row: 2;
		}
	}
</style>
