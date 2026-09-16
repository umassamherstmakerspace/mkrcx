import { describe, expect, it } from 'vitest';
import {
	historyItems,
	printDuration,
	type PrinterEdit,
	type PrinterHistoryData
} from './history-view';

const edit: PrinterEdit = {
	recordedAt: '2026-09-14T12:00:00Z',
	actor: 'user:4',
	version: 1,
	condition: 'out',
	note: 'Fan broken',
	manual: true,
	lifecycle: 'active',
	maintenance: 'repair',
	name: 'Ada'
};
const history = (
	edits: PrinterEdit[] = [],
	events: PrinterHistoryData['events'] = []
): PrinterHistoryData => ({ edits, events, lastSync: null });

describe('printer timeline', () => {
	it('shows outcomes with user and duration, and hides historical starts', () => {
		const base = { recordedAt: edit.recordedAt, detail: '', file: 'part.gcode' };
		const items = historyItems(
			history(
				[],
				[
					{ ...base, sourceId: 'start', eventType: 'started' },
					{
						...base,
						sourceId: 'done',
						eventType: 'printer_completed',
						person: 'Fixture user',
						material: 'PLA',
						durationSeconds: 7510
					}
				]
			)
		);
		expect(items).toHaveLength(1);
		expect(items[0]).toMatchObject({
			title: 'Print completed',
			person: 'Fixture user',
			duration: '2 hr 5 min',
			icon: '✓',
			source: ''
		});
	});
	it('formats old duration-only entries without repeating their minute count', () => {
		const item = historyItems(
			history(
				[],
				[
					{
						sourceId: 'legacy',
						recordedAt: edit.recordedAt,
						eventType: 'printer_cancelled',
						detail: 'Print duration: 125 min'
					}
				]
			)
		)[0];
		expect(item).toMatchObject({ duration: '2 hr 5 min', text: '', icon: '×' });
		expect(printDuration(undefined)).toBeUndefined();
		expect(printDuration(0)).toBe('0 sec');
		expect(printDuration(3600)).toBe('1 hr');
		expect(printDuration(61)).toBe('1 min');
	});
	it('merges notes and automatic events in time order, independently of input order', () => {
		const items = historyItems(
			history(
				[edit, { ...edit, version: 2, recordedAt: '2026-09-16T12:00:00Z', note: 'Fan replaced' }],
				[
					{
						sourceId: 'error',
						recordedAt: '2026-09-15T12:00:00Z',
						eventType: 'printer_error',
						detail: 'Fan failure'
					}
				]
			)
		);
		expect(items.map((item) => item.id)).toEqual(['edit:2', 'event:error', 'edit:1']);
		expect(items[0]).toMatchObject({
			kind: 'note',
			title: 'Note updated',
			text: 'Fan replaced',
			changes: []
		});
		expect(items[1]).toMatchObject({ kind: 'error', source: 'Klipper', text: 'Fan failure' });
	});
	it('describes a lineup change without repeating an unchanged repair note', () => {
		const items = historyItems(
			history([
				edit,
				{ ...edit, version: 2, lifecycle: 'shelved', recordedAt: '2026-09-15T00:00:00Z' }
			])
		);
		expect(items[0]).toMatchObject({
			title: 'Printer updated',
			kind: 'change',
			changes: ['Fleet: In fleet → Shelved']
		});
		expect(items[0].text).toBeUndefined();
	});
	it('shows clearing a note and returning to printer reports as different actions', () => {
		const cleared = { ...edit, version: 2, note: '', recordedAt: '2026-09-15T00:00:00Z' };
		expect(historyItems(history([edit, cleared]))[0].title).toBe('Note cleared');
		const restored = historyItems(history([edit, { ...cleared, manual: false }]))[0];
		expect(restored.changes).toEqual(['Using printer condition & note']);
		expect(restored.title).not.toBe('Note cleared');
	});
	it('does not infer a change across missing or truncated versions', () => {
		const items = historyItems(
			history([
				{ ...edit, version: 49 },
				{ ...edit, version: 51, note: 'New note', recordedAt: '2026-09-16T00:00:00Z' }
			])
		);
		expect(items.map((item) => item.title)).toEqual(['Record saved', 'Record saved']);
		expect(items[0].changes?.some((line) => line.includes('→'))).toBe(false);
	});
	it('preserves multiline staff notes and identifies printer provenance', () => {
		const item = historyItems(
			history(
				[],
				[
					{
						sourceId: 'staff',
						recordedAt: edit.recordedAt,
						eventType: 'staff_runtime_changed',
						detail: 'Condition: out_of_service\nNote: Fan broken.\nReplacement ordered.'
					}
				]
			)
		)[0];
		expect(item).toMatchObject({
			kind: 'note',
			source: 'Printer',
			user: 'Unavailable',
			title: 'Status recorded',
			text: 'Fan broken.\nReplacement ordered.',
			changes: ['Condition: Out of service']
		});
	});
	it('keeps filenames and error details distinct, including missing error messages', () => {
		const item = historyItems(
			history(
				[],
				[
					{
						sourceId: 'failed',
						recordedAt: edit.recordedAt,
						eventType: 'printer_failed',
						detail: 'The station recorded a failed print without an error message.',
						file: 'part.gcode',
						material: 'PLA'
					}
				]
			)
		)[0];
		expect(item).toMatchObject({
			kind: 'error',
			title: 'Print failed',
			text: 'No error message recorded.',
			file: 'part.gcode',
			material: 'PLA'
		});
	});
	it('distinguishes signed-in staff, local PIN and missing station attribution', () => {
		const base = {
			sourceId: 'staff',
			recordedAt: edit.recordedAt,
			eventType: 'staff_runtime_changed',
			detail: 'Condition: working\nNote: Fan replaced'
		};
		expect(
			historyItems(history([], [{ ...base, actorMethod: 'ucard', actorName: 'Alex' }]))[0]
		).toMatchObject({ source: 'Printer', user: 'Alex' });
		expect(historyItems(history([], [{ ...base, actorMethod: 'local_pin' }]))[0].user).toBe(
			'Staff PIN'
		);
		expect(historyItems(history([], [base]))[0].user).toBe('Unavailable');
		expect(historyItems(history([{ ...edit, actorName: 'Alex' }]))[0]).toMatchObject({
			source: 'mkr.cx',
			user: 'Alex'
		});
		expect(
			historyItems(history([{ ...edit, actor: 'service-user:4', actorName: 'Import account' }]))[0]
		).toMatchObject({ source: 'mkr.cx', user: 'Import account' });
	});
	it('accepts an empty collected history', () => {
		expect(historyItems({ events: null, edits: null, lastSync: null })).toEqual([]);
	});
	it('does not mistake an imported human note for an automatic event', () => {
		const item = historyItems(history([{ ...edit, actor: 'service-user:7' }]))[0];
		expect(item.source).toBe('mkr.cx');
		expect(item.kind).toBe('note');
	});
	it('reports maintenance and location changes independently of fleet and condition', () => {
		const items = historyItems(
			history([
				edit,
				{
					...edit,
					version: 2,
					maintenance: 'testing',
					location: 'Repair bench',
					recordedAt: '2026-09-16T12:00:00Z'
				}
			])
		);
		expect(items[0].changes).toEqual([
			'Maintenance: In repair → Testing',
			'Location: Not set → Repair bench'
		]);
		expect(items[0].text).toBeUndefined();
	});
	it('normalizes a legacy repair snapshot without inventing a fleet move', () => {
		const items = historyItems(
			history([
				{ ...edit, lifecycle: 'repair', maintenance: undefined },
				{ ...edit, version: 2, recordedAt: '2026-09-16T12:00:00Z' }
			])
		);
		expect(items[0].changes).toEqual([]);
	});
});

describe('independent history changes', () => {
	const base = {
		sourceId: 'condition:1',
		recordedAt: edit.recordedAt,
		eventType: 'staff_runtime_changed',
		detail: 'Condition: out_of_service\nNote: Waiting for fan',
		previousCondition: 'available'
	};
	it('shows only a changed note, without an unchanged condition', () => {
		expect(
			historyItems(history([], [{ ...base, noteChanged: true, conditionChanged: false }]))[0]
		).toMatchObject({ kind: 'note', title: 'Note updated', text: 'Waiting for fan', changes: [] });
	});
	it('shows a condition-only transition without repeating a saved note', () => {
		const item = historyItems(
			history([], [{ ...base, noteChanged: false, conditionChanged: true }])
		)[0];
		expect(item).toMatchObject({
			title: 'Condition changed',
			changes: ['Condition: Available → Out of service']
		});
		expect(item.text).toBeUndefined();
	});
	it('groups only changes from the same event and handles a cleared note', () => {
		expect(
			historyItems(history([], [{ ...base, noteChanged: true, conditionChanged: true }]))[0].title
		).toBe('Note and condition updated');
		expect(
			historyItems(
				history(
					[],
					[
						{
							...base,
							detail: 'Condition: working\nNote: ',
							noteChanged: true,
							conditionChanged: false
						}
					]
				)
			)[0]
		).toMatchObject({ kind: 'note', title: 'Note cleared', changes: [] });
		expect(
			historyItems(history([], [{ ...base, noteChanged: false, conditionChanged: false }]))
		).toEqual([]);
	});
	it('does not infer which field changed when either comparison is missing', () => {
		expect(historyItems(history([], [{ ...base, noteChanged: false }]))[0].title).toBe(
			'Status recorded'
		);
	});
	it('shows website condition changes and next actions independently', () => {
		expect(
			historyItems(
				history([
					edit,
					{ ...edit, version: 2, condition: 'working', recordedAt: '2026-09-16T12:00:00Z' }
				])
			)[0]
		).toMatchObject({ title: 'Condition changed', changes: ['Condition: Broken → Working'] });
		const next = historyItems(
			history([
				edit,
				{
					...edit,
					version: 2,
					nextAction: 'Verify cable repair',
					recordedAt: '2026-09-16T12:00:00Z'
				}
			])
		)[0];
		expect(next.changes).toEqual(['Next: Verify cable repair']);
		expect(next.text).toBeUndefined();
	});
});
