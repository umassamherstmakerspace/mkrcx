import type { EventInput } from '@fullcalendar/core';
import rosterText from './staff-calendar-roster.txt?raw';

export const COMMITTEE_EVENT_COLOR = '#c2c2c2';
// Anything that is not an exact, recognized staff shift is gray. Unknown
// titles still warn so a newly scheduled staff member can be added explicitly.
export const UNKNOWN_EVENT_COLOR = COMMITTEE_EVENT_COLOR;

export type StaffName = string;
type StaffRosterEntry = Readonly<{ name: StaffName; color: string; emails: readonly string[] }>;

function parseStaffRoster(text: string): StaffRosterEntry[] {
	const entries = text
		.split(/\r?\n/)
		.map((line) => line.trim())
		.filter((line) => line && !line.startsWith('#'))
		.map((line, index) => {
			const [name = '', color = '', emailList = '', ...extra] = line
				.split('|')
				.map((part) => part.trim());
			if (!name || !/^#[0-9a-f]{6}$/i.test(color) || extra.length > 0) {
				throw new Error(`Invalid staff roster line ${index + 1}: ${JSON.stringify(line)}`);
			}
			const emails = emailList
				.split(',')
				.map((email) => email.trim().toLowerCase())
				.filter(Boolean);
			return Object.freeze({ name, color: color.toLowerCase(), emails: Object.freeze(emails) });
		});

	const names = entries.map((entry) => normalizeCalendarTitle(entry.name));
	const emails = entries.flatMap((entry) => entry.emails);
	if (new Set(names).size !== names.length) throw new Error('Duplicate staff calendar label');
	if (new Set(emails).size !== emails.length) throw new Error('Duplicate staff account email');
	return entries;
}

// Edit staff-calendar-roster.txt for semester-to-semester staffing changes.
export const STAFF_ROSTER = Object.freeze(parseStaffRoster(rosterText));
export const STAFF_COLORS: Readonly<Record<StaffName, string>> = Object.freeze(
	Object.fromEntries(STAFF_ROSTER.map(({ name, color }) => [name, color]))
);
export const CURRENT_STAFF_NAMES = Object.freeze(STAFF_ROSTER.map(({ name }) => name));
export const STAFF_NAME_BY_EMAIL: Readonly<Record<string, StaffName>> = Object.freeze(
	Object.fromEntries(
		STAFF_ROSTER.flatMap(({ name, emails }) => emails.map((email) => [email, name] as const))
	)
);

const COLOR_BY_NORMALIZED_NAME: ReadonlyMap<string, string> = new Map(
	CURRENT_STAFF_NAMES.map((name) => [normalizeCalendarTitle(name), STAFF_COLORS[name]])
);

const COMMITTEE_TITLES = new Set(['textiles', '3dp', '3d printing', 'electronics', 'shop']);

export function normalizeCalendarTitle(title: string): string {
	return title.trim().replace(/\s+/g, ' ').toLowerCase();
}

export function isCommitteeMeetingOrNote(title: string): boolean {
	const normalized = normalizeCalendarTitle(title);
	return (
		COMMITTEE_TITLES.has(normalized) ||
		/(?:^|\s)(?:committee|meeting)(?:$|\s|[\u2014\u2013:|/-])/.test(normalized) ||
		/^notes?(?:$|\s|[\u2014\u2013:|/-])/.test(normalized)
	);
}

function textColorFor(backgroundColor: string): '#000000' | '#ffffff' {
	const channels = [1, 3, 5].map((index) =>
		Number.parseInt(backgroundColor.slice(index, index + 2), 16)
	);
	const luminance = channels
		.map((channel) => channel / 255)
		.map((channel) =>
			channel <= 0.04045 ? channel / 12.92 : Math.pow((channel + 0.055) / 1.055, 2.4)
		)
		.reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0);

	const whiteContrast = 1.05 / (luminance + 0.05);
	const blackContrast = (luminance + 0.05) / 0.05;
	return blackContrast >= whiteContrast ? '#000000' : '#ffffff';
}

export function createStaffCalendarColorizer(
	onUnknownTitle: (title: string) => void = (title) =>
		console.warn(`Unmapped staff calendar title: ${JSON.stringify(title)}`)
): (event: EventInput) => EventInput {
	const warnedTitles = new Set<string>();

	return (event: EventInput): EventInput => {
		const title = event.title ?? '';
		const normalized = normalizeCalendarTitle(title);
		let backgroundColor: string | undefined = COLOR_BY_NORMALIZED_NAME.get(normalized);

		if (!backgroundColor && isCommitteeMeetingOrNote(title)) {
			backgroundColor = COMMITTEE_EVENT_COLOR;
		}

		if (!backgroundColor) {
			backgroundColor = UNKNOWN_EVENT_COLOR;
			if (!warnedTitles.has(normalized)) {
				warnedTitles.add(normalized);
				onUnknownTitle(title);
			}
		}

		return {
			...event,
			backgroundColor,
			borderColor: backgroundColor,
			textColor: textColorFor(backgroundColor)
		};
	};
}

export const colorizeStaffCalendarEvent = createStaffCalendarColorizer();
