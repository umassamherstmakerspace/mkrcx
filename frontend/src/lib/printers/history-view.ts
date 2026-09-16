export type PrinterEvent = {
	sourceId: string;
	recordedAt: string;
	eventType: string;
	detail: string;
	file?: string;
	material?: string;
};
export type PrinterEdit = {
	recordedAt: string;
	actor: string;
	version: number;
	condition: string;
	note: string;
	manual: boolean;
	lifecycle: string;
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
};

const labels: Record<string, string> = {
	working: 'Working',
	limited: 'Limited use',
	limited_use: 'Limited use',
	out: 'Out of service',
	out_of_service: 'Out of service',
	unknown: 'Unknown',
	active: 'Active',
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
	const item: HistoryItem = {
		id: `event:${event.sourceId}`,
		recordedAt: event.recordedAt,
		kind:
			event.eventType === 'printer_error' || event.eventType === 'printer_failed' ? 'error' : 'job',
		title: eventLabels[event.eventType] ?? event.eventType.replaceAll('_', ' '),
		source: 'Automatic',
		file: event.file,
		material: event.material,
		text:
			event.detail === 'The station recorded a failed print without an error message.'
				? 'No error message recorded.'
				: event.detail
	};
	if (['staff_runtime_changed', 'printer_runtime_changed'].includes(event.eventType)) {
		const condition = event.detail.match(/^Condition: ([^\n]*)/);
		const note = event.detail.match(/(?:^|\n)Note: ([\s\S]*)/);
		item.kind = note?.[1] ? 'note' : 'change';
		item.title = note?.[1] ? 'Note & condition' : 'Condition updated';
		item.source = event.eventType === 'staff_runtime_changed' ? 'Staff · Printer' : 'Automatic';
		item.text = note ? note[1] : condition ? undefined : event.detail;
		item.changes = condition ? [`Condition: ${label(condition[1])}`] : [];
	}
	return item;
}

export function historyItems(history: PrinterHistoryData): HistoryItem[] {
	const edits = [...(history.edits ?? [])].sort((a, b) => a.version - b.version);
	const items = (history.events ?? []).map(eventItem);
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
				changes.push(`Lineup: ${label(previous.lifecycle)} → ${label(edit.lifecycle)}`);
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
			changes.push(`Lineup: ${label(edit.lifecycle)}`);
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
				? 'Staff · mkr.cx'
				: edit.actor.startsWith('service-user:')
					? 'Automatic · mkr.cx'
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
