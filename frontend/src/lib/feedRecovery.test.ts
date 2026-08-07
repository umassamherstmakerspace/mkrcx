import { describe, expect, it } from 'vitest';
import type { LeashFeedItem } from '$lib/leash';
import { applyFeedRetention, FeedRecoveryBuffer, formatFeedTimestamp } from '$lib/feedRecovery';

function items(first: number, last: number): LeashFeedItem[] {
	return Array.from({ length: last - first + 1 }, (_, offset) => ({
		ID: first + offset,
		CreatedAt: '2026-08-03T00:00:00Z',
		FeedID: 1,
		AddedBy: 1,
		LogLevel: 1,
		UserID: 0,
		Title: 'Tap',
		Message: 'Allowed'
	}));
}

describe('FeedRecoveryBuffer', () => {
	it('does not let an early WebSocket item skip HTTP catch-up pages', () => {
		const recovery = new FeedRecoveryBuffer();
		let visible = recovery.mergeHTTP([], items(5, 5));

		visible = recovery.mergeSocket(visible, items(205, 205));
		expect(recovery.cursor).toBe(5);

		const requestedAfter = [recovery.cursor];
		visible = recovery.mergeHTTP(visible, items(6, 105));
		requestedAfter.push(recovery.cursor);
		visible = recovery.mergeHTTP(visible, items(106, 205));

		expect(requestedAfter).toEqual([5, 105]);
		expect(recovery.cursor).toBe(205);
		expect(visible.map((item) => item.ID)).toEqual(
			items(106, 205)
				.reverse()
				.map((item) => item.ID)
		);
	});
});

describe('formatFeedTimestamp', () => {
	it('includes both the calendar date and time', () => {
		const formatted = formatFeedTimestamp(new Date('2026-08-07T14:05:06Z'), 'en-US');
		expect(formatted).toContain('2026');
		expect(formatted).toContain('Aug');
		expect(formatted).toMatch(/7/);
		expect(formatted).toMatch(/\d{1,2}:05:06/);
	});
});

describe('applyFeedRetention', () => {
	it('removes seven-day-old signin rows without changing other feeds', () => {
		const now = new Date('2026-08-07T15:00:00Z').getTime();
		const feedItems = [
			...items(1, 1).map((item) => ({ ...item, CreatedAt: '2026-07-31T14:59:59Z' })),
			...items(2, 2).map((item) => ({ ...item, CreatedAt: '2026-08-01T15:00:00Z' }))
		];

		expect(applyFeedRetention(feedItems, 'signin', now).map((item) => item.ID)).toEqual([2]);
		expect(applyFeedRetention(feedItems, 'other-feed', now)).toEqual(feedItems);
	});
});
