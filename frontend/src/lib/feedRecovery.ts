import type { LeashFeedItem } from '$lib/leash';

const checkinFeedRetentionMilliseconds = 7 * 24 * 60 * 60 * 1000;

export function formatFeedTimestamp(value: string | Date, locales?: string | string[]): string {
	const date = value instanceof Date ? value : new Date(value);
	return new Intl.DateTimeFormat(locales, {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
		second: '2-digit'
	}).format(date);
}

export function applyFeedRetention(
	items: LeashFeedItem[],
	feedName: string | undefined,
	now = Date.now()
): LeashFeedItem[] {
	if (feedName !== 'signin') return items;
	const cutoff = now - checkinFeedRetentionMilliseconds;
	return items.filter((item) => new Date(item.CreatedAt).getTime() >= cutoff);
}

function mergeFeedItems(current: LeashFeedItem[], incoming: LeashFeedItem[]): LeashFeedItem[] {
	const byID = new Map(current.map((item) => [item.ID, item]));
	for (const item of incoming) byID.set(item.ID, item);
	return [...byID.values()].sort((a, b) => b.ID - a.ID).slice(0, 100);
}

// Only database-backed HTTP pages advance this cursor. WebSocket items can be
// newer than an in-flight catch-up page and must never move the recovery point.
export class FeedRecoveryBuffer {
	cursor = 0;

	mergeHTTP(current: LeashFeedItem[], incoming: LeashFeedItem[]): LeashFeedItem[] {
		for (const item of incoming) this.cursor = Math.max(this.cursor, item.ID);
		return mergeFeedItems(current, incoming);
	}

	mergeSocket(current: LeashFeedItem[], incoming: LeashFeedItem[]): LeashFeedItem[] {
		return mergeFeedItems(current, incoming);
	}
}
