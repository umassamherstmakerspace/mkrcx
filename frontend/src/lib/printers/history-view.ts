import { printerStates, printRecipient } from './printer-state';
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
	noteChanged?: boolean;
	conditionChanged?: boolean;
	previousCondition?: string;
};
export type PrinterEdit = {
	recordedAt: string;
	actor: string;
	actorName?: string;
	version: number;
	condition: string;
	note: string;
	nextAction?: string;
	manual: boolean;
	lifecycle: string;
	maintenance?: string;
	location?: string;
	name: string;
};
export type PrinterHistoryData = {
	historical?: HistoricalEntry[];
	pageIds?: string[];
	nextCursor?: string;
	legacy?: { jobs: number; first: string | null };
	meters?: HistoricalEntry[];
	origins?: HistoricalEntry[];
	summaries?:
		| {
				sourceId: string;
				reportDate: string;
				body: string;
				sources?: { url: string; label: string }[] | null;
				preparedBy: string;
				importedAt: string;
		  }[]
		| null;
	usage?: { seconds: number; jobs: number; missingDurations: number; firstOutcome: string | null };
	events: PrinterEvent[] | null;
	edits: PrinterEdit[] | null;
	lastSync: string | null;
};
export type HistoricalEntry = {
	sourceId: string;
	recordedAt: string;
	dateOnly: boolean;
	kind: 'submission' | 'service' | 'report' | 'meter' | 'origin' | 'retirement';
	body: string;
	reporter?: string;
	preparedBy?: string;
	person?: string;
	file?: string;
	material?: string;
	meterHours?: number;
	estimatedSeconds?: number;
};
export type HistoryItem = {
	id: string;
	recordedAt: string;
	kind: 'note' | 'job' | 'error' | 'change' | 'summary';
	dateOnly?: boolean;
	links?: { url: string; label: string }[];
	preparedBy?: string;
	printOutcome?: boolean;
	title: string;
	source: string;
	actor?: string;
	user?: string;
	automatic?: boolean;
	text?: string;
	changes?: string[];
	file?: string;
	material?: string;
	person?: string;
	duration?: string;
	icon?: string;
	outcome?: 'completed' | 'cancelled' | 'failed' | 'unknown';
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
	available: 'Available',
	needs_attention: 'Limited use',
	limited: 'Limited use',
	limited_use: 'Limited use',
	out: 'Broken',
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
	printer_error: 'Printer error',
	printer_responding: 'Printer responding again'
};

function eventItem(event: PrinterEvent): HistoryItem | null {
	const legacyDuration = event.detail.match(/(?:^|\n)Print duration: (\d+) min(?:\n|$)/);
	const detail = event.detail.replace(/(?:^|\n)Print duration: \d+ min(?=\n|$)/, '').trim();
	const item: HistoryItem = {
		outcome:
			event.eventType === 'printer_completed'
				? 'completed'
				: event.eventType === 'printer_cancelled'
					? 'cancelled'
					: event.eventType === 'printer_failed'
						? 'failed'
						: undefined,
		printOutcome: ['printer_completed', 'printer_cancelled', 'printer_failed'].includes(
			event.eventType
		),
		id: `event:${event.sourceId}`,
		automatic: ['printer_error', 'printer_responding'].includes(event.eventType),
		recordedAt: event.recordedAt,
		kind:
			event.eventType === 'printer_error' || event.eventType === 'printer_failed' ? 'error' : 'job',
		title: eventLabels[event.eventType] ?? event.eventType.replaceAll('_', ' '),
		source: ['printer_error', 'printer_responding'].includes(event.eventType) ? 'Klipper' : '',
		file: event.file,
		material: event.material,
		person: printRecipient(event.person),
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
	if (event.eventType === 'printer_responding') {
		item.kind = 'change';
		item.icon = '·';
	}
	if (
		['staff_runtime_changed', 'printer_runtime_changed', 'system_runtime_changed'].includes(
			event.eventType
		)
	) {
		const condition = event.detail.match(/^Condition: ([^\n]*)/);
		const note = event.detail.match(/(?:^|\n)Note: ([\s\S]*)/);
		if (event.noteChanged === false && event.conditionChanged === false) return null;
		const known = event.noteChanged !== undefined && event.conditionChanged !== undefined;
		const showNote = known ? event.noteChanged : !!note;
		const showCondition = known ? event.conditionChanged : !!condition;
		item.kind = showNote ? 'note' : 'change';
		item.title = !known
			? 'Status recorded'
			: event.noteChanged
				? event.conditionChanged
					? 'Note and condition updated'
					: note?.[1]
						? 'Note updated'
						: 'Note cleared'
				: 'Condition changed';
		item.source = 'Printer';
		item.automatic =
			event.eventType !== 'staff_runtime_changed' || event.actorMethod === 'printer_api';
		item.user = item.automatic
			? undefined
			: event.actorMethod === 'local_pin'
				? 'Staff PIN'
				: event.actorMethod === 'ucard'
					? `${event.actorName || 'Attribution unavailable'} · Card tap`
					: event.actorName || 'Attribution unavailable';
		item.icon = undefined;
		item.text =
			showNote && note ? note[1] || undefined : !note && !condition ? event.detail : undefined;
		item.changes =
			showCondition && condition
				? [
						`Condition: ${known && event.previousCondition ? label(event.previousCondition) + ' → ' : ''}${label(condition[1])}`
					]
				: [];
	}
	return item;
}

export function historyItems(history: PrinterHistoryData): HistoryItem[] {
	const edits = (history.edits ?? [])
		.map((edit) => ({ ...edit, ...printerStates(edit) }))
		.sort((a, b) => a.version - b.version);
	const items = (history.events ?? [])
		.filter((event) => event.eventType !== 'started')
		.map(eventItem)
		.filter((item): item is HistoryItem => item !== null);
	for (const entry of history.historical ?? []) {
		const submission = entry.kind === 'submission';
		items.push({
			id: `historical:${entry.sourceId}`,
			recordedAt: entry.recordedAt,
			dateOnly: entry.dateOnly,
			kind: submission ? 'job' : entry.kind === 'meter' ? 'change' : 'summary',
			printOutcome: submission,
			outcome: submission ? 'unknown' : undefined,
			title: submission ? 'Print logged · outcome unknown' : '',
			source: '',
			text: entry.body,
			user: entry.reporter,
			preparedBy: entry.preparedBy,
			person: entry.person,
			file: entry.file,
			material: entry.material,
			duration:
				entry.estimatedSeconds === undefined
					? undefined
					: `${printDuration(entry.estimatedSeconds)} estimated`,
			icon: submission ? '·' : undefined
		});
	}
	for (const summary of history.summaries ?? []) {
		items.push({
			id: `summary:${summary.sourceId}`,
			recordedAt: `${summary.reportDate}T00:00:00Z`,
			dateOnly: true,
			kind: 'summary',
			title: '',
			source: '',
			text: summary.body,
			preparedBy: summary.preparedBy,
			links: (summary.sources ?? []).filter((source) => {
				try {
					return new URL(source.url).protocol === 'https:';
				} catch {
					return false;
				}
			})
		});
	}
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
			if ((previous.nextAction ?? '') !== (edit.nextAction ?? ''))
				changes.push(edit.nextAction ? `Next: ${edit.nextAction}` : 'Next action cleared');
			if (previous.manual !== edit.manual)
				changes.push(edit.manual ? 'Assessment saved' : 'Using printer condition & note');
			if (edit.manual && previous.condition !== edit.condition)
				changes.push(`Condition: ${label(previous.condition)} → ${label(edit.condition)}`);
			if (edit.manual && previous.note !== edit.note) {
				title =
					previous.condition !== edit.condition
						? 'Note and condition updated'
						: edit.note
							? 'Note updated'
							: 'Note cleared';
				text = edit.note || undefined;
			} else if (changes.length)
				title =
					changes.length === 1 && changes[0].startsWith('Condition:')
						? 'Condition changed'
						: 'Printer updated';
		} else {
			// A saved snapshot is not evidence that its note was changed at this time.
			changes.push(`Fleet: ${label(edit.lifecycle)}`);
			if (edit.nextAction) changes.push(`Next: ${edit.nextAction}`);
			if (edit.maintenance !== 'none') changes.push(`Maintenance: ${label(edit.maintenance)}`);
			if (edit.manual) {
				changes.push(`Condition: ${label(edit.condition)}`);
				text = edit.note || undefined;
			} else changes.push('Using printer condition & note');
		}
		items.push({
			id: `edit:${edit.version}`,
			recordedAt: edit.recordedAt,
			kind: text || title.startsWith('Note') ? 'note' : 'change',
			title,
			source: 'mkr.cx',
			user:
				edit.actorName || (edit.actor.startsWith('user:') ? 'Attribution unavailable' : undefined),
			actor: edit.actor,
			text,
			changes
		});
	}
	const pageOrder = new Map(history.pageIds?.map((id, index) => [id, index]));
	return items
		.filter((item) => !history.pageIds || history.pageIds.includes(item.id))
		.sort((a, b) =>
			history.pageIds
				? pageOrder.get(a.id)! - pageOrder.get(b.id)!
				: Date.parse(b.recordedAt) - Date.parse(a.recordedAt) || a.id.localeCompare(b.id)
		);
}

export type HistoryFilter = 'all' | 'updates' | 'prints';

export function filterHistoryItems(items: HistoryItem[], filter: HistoryFilter): HistoryItem[] {
	return items.filter((item) =>
		filter === 'all'
			? true
			: filter === 'prints'
				? item.printOutcome
				: ['note', 'error', 'summary'].includes(item.kind)
	);
}
