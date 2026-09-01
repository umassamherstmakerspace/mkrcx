import { describe, expect, it, vi } from 'vitest';
import {
	COMMITTEE_EVENT_COLOR,
	CURRENT_STAFF_NAMES,
	STAFF_COLORS,
	UNKNOWN_EVENT_COLOR,
	createStaffCalendarColorizer
} from '$lib/staffCalendarColors';

describe('staff calendar color policy', () => {
	it('assigns every current person the expected unique non-gray color', () => {
		expect(STAFF_COLORS).toEqual({
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
		expect(CURRENT_STAFF_NAMES).toHaveLength(19);
		expect(new Set(Object.values(STAFF_COLORS)).size).toBe(19);
		expect(Object.values(STAFF_COLORS)).not.toContain(COMMITTEE_EVENT_COLOR);
	});

	it('normalizes case and whitespace without changing a person color', () => {
		const colorize = createStaffCalendarColorizer();
		expect(colorize({ title: 'Niall' }).backgroundColor).toBe(STAFF_COLORS.Niall);
		expect(colorize({ title: '  NIALL\t' }).backgroundColor).toBe(STAFF_COLORS.Niall);
		expect(colorize({ title: 'julius' }).backgroundColor).toBe(STAFF_COLORS.Julius);
	});

	it.each([
		'Textiles',
		'3DP',
		'3D Printing',
		'Electronics',
		'Shop',
		'Textiles Committee',
		'Team meeting',
		'Notes — fall schedule'
	])('renders %s in committee gray', (title) => {
		const event = createStaffCalendarColorizer()({ title });
		expect(event.backgroundColor).toBe(COMMITTEE_EVENT_COLOR);
	});

	it('classifies only from the title, not from a shift description', () => {
		const event = createStaffCalendarColorizer()({
			title: 'Shira',
			description: 'Attending a committee later'
		});
		expect(event.backgroundColor).toBe(STAFF_COLORS.Shira);
	});

	it('renders tours and other non-shifts gray while warning once per unknown title', () => {
		const onUnknown = vi.fn();
		const colorize = createStaffCalendarColorizer(onUnknown);
		expect(colorize({ title: 'Transfer students quick tour' }).backgroundColor).toBe(
			COMMITTEE_EVENT_COLOR
		);
		expect(colorize({ title: '  transfer   students quick tour ' }).backgroundColor).toBe(
			UNKNOWN_EVENT_COLOR
		);
		expect(onUnknown).toHaveBeenCalledOnce();
		expect(onUnknown).toHaveBeenCalledWith('Transfer students quick tour');
	});

	it('changes only display-color fields', () => {
		const original = {
			title: 'Brooke',
			description: 'unchanged',
			start: '2026-09-08T10:00:00-04:00',
			end: '2026-09-08T13:00:00-04:00',
			allDay: false,
			id: 'calendar-uid',
			uid: 'calendar-uid',
			sequence: 4,
			recurrenceId: '2026-09-08T10:00:00-04:00',
			extendedProps: { source: 'Team calendar' }
		};
		const colored = createStaffCalendarColorizer()(original);
		const { backgroundColor, borderColor, textColor, ...preserved } = colored;

		expect(preserved).toEqual(original);
		expect(backgroundColor).toBe(STAFF_COLORS.Brooke);
		expect(borderColor).toBe(STAFF_COLORS.Brooke);
		expect(textColor).toMatch(/^#[0-9a-f]{6}$/);
	});
});
