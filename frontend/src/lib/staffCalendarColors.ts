import type { EventInput } from '@fullcalendar/core';

export const COMMITTEE_EVENT_COLOR = '#c2c2c2';
// Anything that is not an exact, recognized staff shift is gray. Unknown
// titles still warn so a newly scheduled staff member can be added explicitly.
export const UNKNOWN_EVENT_COLOR = COMMITTEE_EVENT_COLOR;

// Frozen from the Fall 2026 Team-calendar overlap plan. Keep this explicit so a
// new person cannot silently take an existing person's color.
export const STAFF_COLORS = Object.freeze({
	Shira: '#9fc6e7',
	Lauren: '#42d692',
	Keegan: '#f83a22',
	Jack: '#a47ae2',
	Niall: '#cd74e6',
	Julius: '#4986e7',
	Tobias: '#f691b2',
	Peyton: '#92e1c0',
	Brody: '#fbe983',
	Duy: '#b99aff',
	Punya: '#fa573c',
	Brooke: '#ff7537',
	Rigel: '#b3dc6c',
	Sean: '#ffad46',
	Quinlan: '#fad165',
	Marcelo: '#9a9cff',
	Sastha: '#7bd148',
	Ethan: '#16a765',
	Tvisha: '#9fe1e7'
});

export type StaffName = keyof typeof STAFF_COLORS;
export const CURRENT_STAFF_NAMES = Object.freeze(Object.keys(STAFF_COLORS) as StaffName[]);

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
