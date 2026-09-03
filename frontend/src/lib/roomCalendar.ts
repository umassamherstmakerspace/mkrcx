import type { EventInput } from '@fullcalendar/core';

const BUSY_COLOR = '#64748b';
const STAFF_COLOR = '#7c3aed';

export function presentRoomCalendarEvent(event: EventInput, isStaff: boolean): EventInput {
	if (isStaff) {
		return {
			...event,
			backgroundColor: STAFF_COLOR,
			borderColor: STAFF_COLOR
		};
	}

	return {
		title: 'Busy',
		start: event.start,
		end: event.end,
		allDay: event.allDay,
		backgroundColor: BUSY_COLOR,
		borderColor: BUSY_COLOR
	};
}
