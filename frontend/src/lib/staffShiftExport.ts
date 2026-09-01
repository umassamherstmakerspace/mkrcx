import type { EventInput } from '@fullcalendar/core';
import {
	CURRENT_STAFF_NAMES,
	normalizeCalendarTitle,
	type StaffName
} from '$lib/staffCalendarColors';

type StaffIdentity = {
	email: string;
	name: string;
};

// Most accounts match the calendar through the first word of the Leash name.
// Add only exceptions here, keyed by lower-case account email. This table is
// intentionally separate from authentication and grants no additional access.
export const STAFF_SHIFT_NAME_OVERRIDES: Readonly<Record<string, StaffName>> = Object.freeze({});

export function resolveStaffShiftName(
	user: StaffIdentity,
	overrides: Readonly<Record<string, StaffName>> = STAFF_SHIFT_NAME_OVERRIDES
): StaffName | undefined {
	const override = overrides[user.email.trim().toLowerCase()];
	if (override) return override;

	const firstName = user.name.trim().split(/\s+/)[0] || '';
	const normalized = normalizeCalendarTitle(firstName);
	return CURRENT_STAFF_NAMES.find((name) => normalizeCalendarTitle(name) === normalized);
}

function eventDate(value: EventInput['start'] | EventInput['end']): Date | undefined {
	if (value === undefined || value === null) return undefined;
	const parsed = value instanceof Date ? new Date(value) : new Date(value as string | number);
	return Number.isFinite(parsed.getTime()) ? parsed : undefined;
}

function utcStamp(date: Date): string {
	return date
		.toISOString()
		.replace(/[-:]/g, '')
		.replace(/\.\d{3}Z$/, 'Z');
}

export function escapeIcsText(value: string): string {
	return value
		.replace(/\\/g, '\\\\')
		.replace(/\r\n|\r|\n/g, '\\n')
		.replace(/;/g, '\\;')
		.replace(/,/g, '\\,');
}

export function foldIcsLine(line: string): string {
	const encoder = new TextEncoder();
	const chunks: string[] = [];
	let current = '';

	for (const character of line) {
		const limit = chunks.length === 0 ? 75 : 74;
		if (current && encoder.encode(current + character).length > limit) {
			chunks.push(current);
			current = character;
		} else {
			current += character;
		}
	}
	chunks.push(current);
	return chunks.join('\r\n ');
}

function sourceUid(event: EventInput, name: StaffName, start: Date): string {
	const record = event as EventInput & { uid?: unknown };
	const source =
		typeof record.uid === 'string' && record.uid.trim()
			? record.uid.trim()
			: typeof event.id === 'string' && event.id.trim()
				? event.id.trim()
				: `${normalizeCalendarTitle(name)}-${utcStamp(start)}`;
	return `${source}-${utcStamp(start)}@mkr.cx`;
}

export function staffShiftEvents(events: EventInput[], name: StaffName): EventInput[] {
	const normalizedName = normalizeCalendarTitle(name);
	return events.filter((event) => {
		if (event.allDay || normalizeCalendarTitle(event.title ?? '') !== normalizedName) return false;
		const start = eventDate(event.start);
		const end = eventDate(event.end);
		return Boolean(start && end && end > start);
	});
}

export function buildStaffShiftCalendar(
	name: StaffName,
	events: EventInput[],
	generatedAt = new Date()
): string {
	const lines = [
		'BEGIN:VCALENDAR',
		'VERSION:2.0',
		'PRODID:-//UMass Amherst Makerspace//Staff Shift Export//EN',
		'CALSCALE:GREGORIAN',
		'METHOD:PUBLISH',
		`X-WR-CALNAME:${escapeIcsText(`${name}'s Makerspace shifts`)}`
	];

	for (const event of staffShiftEvents(events, name)) {
		const start = eventDate(event.start);
		const end = eventDate(event.end);
		if (!start || !end) continue;

		lines.push(
			'BEGIN:VEVENT',
			`UID:${escapeIcsText(sourceUid(event, name, start))}`,
			`DTSTAMP:${utcStamp(generatedAt)}`,
			`DTSTART:${utcStamp(start)}`,
			`DTEND:${utcStamp(end)}`,
			'SUMMARY:UMass Makerspace shift',
			'LOCATION:UMass Amherst Makerspace',
			`DESCRIPTION:${escapeIcsText(`Scheduled staff shift for ${name}.`)}`,
			'URL:https://mkr.cx/staff/schedule',
			'END:VEVENT'
		);
	}

	lines.push('END:VCALENDAR');
	return `${lines.map(foldIcsLine).join('\r\n')}\r\n`;
}

export function staffShiftFilename(name: StaffName): string {
	return `makerspace-${normalizeCalendarTitle(name).replace(/[^a-z0-9]+/g, '-')}-shifts.ics`;
}
