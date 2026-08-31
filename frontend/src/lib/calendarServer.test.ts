import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const calendarMock = vi.hoisted(() => ({
	between: vi.fn(() => []),
	cleanAndParse: vi.fn()
}));

vi.mock('$lib/calendar', () => ({
	CalendarSet: {
		cleanAndParse: calendarMock.cleanAndParse
	}
}));

import { CalendarServer } from '$lib/calendarServer';

describe('CalendarServer source cache', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-09-08T12:00:00Z'));
		calendarMock.between.mockClear();
		calendarMock.cleanAndParse.mockReset();
		calendarMock.cleanAndParse.mockReturnValue({ between: calendarMock.between });
	});

	afterEach(() => vi.useRealTimers());

	it('reuses the private ICS read for five minutes and then refreshes it', async () => {
		const sourceFetch = vi.fn(async () => new Response('private ICS contents'));
		const calendar = new CalendarServer(
			'https://calendar.example/private.ics',
			sourceFetch as typeof globalThis.fetch
		);
		const start = new Date('2026-09-08T00:00:00Z');
		const end = new Date('2026-09-09T00:00:00Z');

		await calendar.getEventsBetween(start, end);
		vi.advanceTimersByTime(4 * 60 * 1000 + 59 * 1000);
		await calendar.getEventsBetween(start, end);
		expect(sourceFetch).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(2 * 1000);
		await calendar.getEventsBetween(start, end);
		expect(sourceFetch).toHaveBeenCalledTimes(2);
		expect(calendarMock.cleanAndParse).toHaveBeenCalledTimes(2);
	});
});
