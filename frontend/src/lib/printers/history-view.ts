import { printerStates } from './printer-state';
export type PrinterEvent = {
	sourceId: string;
	recordedAt: string;
	eventType: string;
	detail: string;
	file?: string;
	material?: string;
	person?: string;
	durationSeconds?: number;
	actorName?: string;
	actorMethod?: 'ucard' | 'local_pin' | 'printer_api';
};
export type PrinterEdit = {
	recordedAt: string;
	actor: string;
	actorName?: string;
	version: number;
	condition: string;
	note: string;
	manual: boolean;
	lifecycle: string;
	maintenance?: string;
	location?: string;
	name: string;
};
export type PrinterHistoryData = {
	events: PrinterEvent[] | null;
	edits: PrinterEdit[] | null;
	lastSync: string | null;
};
export type HistoryItem = {
	id: string;
	recordedAt: string;
	kind: 'note' | 'job' | 'error' | 'change';
	title: string;
	source: string;
	actor?: string;
	text?: string;
	changes?: string[];
	file?: string;
	material?: string;
	person?: string;
	duration?: string;
	icon?: string;
};

export function printDuration(seconds: number | undefined): string | undefined {
	if (seconds === undefined || !Number.isFinite(seconds) || seconds < 0) return undefined;
	if (seconds < 60) return `${Math.round(seconds)} sec`;
	const minutes = Math.round(seconds / 60);
	const hours = Math.floor(minutes / 60);
	return [hours ? `${hours} hr` : '', minutes % 60 ? `${minutes % 60} min` : '']
		.filter(Boolean)
		.join(' ');
}

const labels: Record<string, string> = {
	working: 'Working',
	limited: 'Limited use',
	limited_use: 'Limited use',
	out: 'Out of service',
	out_of_service: 'Out of service',
	unknown: 'Unknown',
	active: 'In fleet',
	none: 'None',
	diagnosis: 'Needs diagnosis',
	testing: 'Testing',
	repair: 'In repair',
	shelved: 'Shelved',
	retired: 'Retired'
};
const label = (value: string) => labels[value] ?? value;
const eventLabels: Record<string, string> = {
	started: 'Print started',
	printer_completed: 'Print completed',
	printer_cancelled: 'Print cancelled',
	printer_failed: 'Print failed',
	printer_error: 'Printer error'
};

function eventItem(event: PrinterEvent): HistoryItem {
	const legacyDuration = event.detail.match(/(?:^|\n)Print duration: (\d+) min(?:\n|$)/);
	const detail = event.detail.replace(/(?:^|\n)Print duration: \d+ min(?=\n|$)/, '').trim();
	const item: HistoryItem = {
		id: `event:${event.sourceId}`,
		recordedAt: event.recordedAt,
		kind:
			event.eventType === 'printer_error' || event.eventType === 'printer_failed' ? 'error' : 'job',
		title: eventLabels[event.eventType] ?? event.eventType.replaceAll('_', ' '),
		source: event.eventType === 'printer_error' ? 'Klipper' : '',
		file: event.file,
		material: event.material,
		person: event.person,
		duration: printDuration(
			event.durationSeconds ?? (legacyDuration ? Number(legacyDuration[1]) * 60 : undefined)
		),
		icon:
			event.eventType === 'printer_completed'
				? '✓'
				: event.eventType === 'printer_cancelled'
					? '×'
					: '!',
		text:
			detail === 'The station recorded a failed print without an error message.'
				? 'No error message recorded.'
				: detail
	};
	if (
		['staff_runtime_changed', 'printer_runtime_changed', 'system_runtime_changed'].includes(
			event.eventType
		)
	) {
		const condition = event.detail.match(/^Condition: ([^\n]*)/);
		const note = event.detail.match(/(?:^|\n)Note: ([\s\S]*)/);
		item.kind = note?.[1] ? 'note' : 'change';
		item.title = note?.[1] ? 'Note & condition' : 'Condition updated';
		item.source =
			event.eventType === 'staff_runtime_changed'
				? `${event.actorMethod === 'local_pin' ? 'Local PIN' : event.actorName || (event.actorMethod === 'ucard' ? 'Staff card · Name not recorded' : 'Staff identity not recorded')} · Printer`
				: 'Printer station';
		item.icon = undefined;
		item.text = note ? note[1] : condition ? undefined : event.detail;
		item.changes = condition ? [`Condition: ${label(condition[1])}`] : [];
	}
	return item;
}

export function historyItems(history: PrinterHistoryData): HistoryItem[] {
	const edits = (history.edits ?? [])
		.map((edit) => ({ ...edit, ...printerStates(edit) }))
		.sort((a, b) => a.version - b.version);
	const items = (history.events ?? [])
		.filter((event) => event.eventType !== 'started')
		.map(eventItem);
	for (let i = 0; i < edits.length; i++) {
		const edit = edits[i];
		// The API returns a bounded window. Only compare consecutive saved versions.
		const previous = edits[i - 1]?.version === edit.version - 1 ? edits[i - 1] : undefined;
		const changes: string[] = [];
		let text: string | undefined;
		let title = 'Record saved';
		if (previous) {
			if (previous.name !== edit.name) changes.push(`Name: ${previous.name} → ${edit.name}`);
			if (previous.lifecycle !== edit.lifecycle)
				changes.push(`Fleet: ${label(previous.lifecycle)} → ${label(edit.lifecycle)}`);
			if (previous.maintenance !== edit.maintenance)
				changes.push(`Maintenance: ${label(previous.maintenance)} → ${label(edit.maintenance)}`);
			if ((previous.location ?? '') !== (edit.location ?? ''))
				changes.push(`Location: ${previous.location || 'Not set'} → ${edit.location || 'Not set'}`);
			if (previous.manual !== edit.manual)
				changes.push(
					edit.manual ? 'Condition & note saved here' : 'Using printer condition & note'
				);
			if (edit.manual && previous.condition !== edit.condition)
				changes.push(`Condition: ${label(previous.condition)} → ${label(edit.condition)}`);
			if (edit.manual && previous.note !== edit.note) {
				title = edit.note ? 'Note updated' : 'Note cleared';
				text = edit.note || undefined;
			} else if (changes.length) title = 'Printer updated';
		} else {
			// A saved snapshot is not evidence that its note was changed at this time.
			changes.push(`Fleet: ${label(edit.lifecycle)}`);
			if (edit.maintenance !== 'none') changes.push(`Maintenance: ${label(edit.maintenance)}`);
			if (edit.manual) {
				changes.push(`Condition: ${label(edit.condition)}`);
				text = edit.note || undefined;
			} else changes.push('Using printer condition & note');
		}
		items.push({
			id: `edit:${edit.version}`,
			recordedAt: edit.recordedAt,
			kind: text ? 'note' : 'change',
			title,
			source: edit.actor.startsWith('user:')
				? `${edit.actorName || 'Staff name not recorded'} · mkr.cx`
				: edit.actorName
					? `${edit.actorName} · API · mkr.cx`
					: 'mkr.cx',
			actor: edit.actor,
			text,
			changes
		});
	}
	return items.sort(
		(a, b) => Date.parse(b.recordedAt) - Date.parse(a.recordedAt) || a.id.localeCompare(b.id)
	);
}
