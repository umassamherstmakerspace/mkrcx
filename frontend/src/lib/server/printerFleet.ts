import { timingSafeEqual } from 'node:crypto';
export function tokenMatches(supplied: string | null, expected: string | undefined): boolean {
	if (!expected || !supplied?.startsWith('Bearer ')) return false;
	const actual = Buffer.from(supplied.slice(7));
	const wanted = Buffer.from(expected);
	return actual.length === wanted.length && timingSafeEqual(actual, wanted);
}
export async function readFleet(fetch: typeof globalThis.fetch, endpoint: string, token?: string) {
	const base = `${endpoint.replace(/\/$/, '')}/api/printer-fleet`;
	if (token) {
		const staff = await fetch(`${base}/staff`, {
			headers: { Authorization: `Bearer ${token}` },
			signal: AbortSignal.timeout(5000)
		});
		if (staff.ok) return staff.json();
		if (![401, 403].includes(staff.status)) throw new Error('Printer registry unavailable');
	}
	const response = await fetch(base, { signal: AbortSignal.timeout(5000) });
	if (!response.ok) throw new Error('Printer registry unavailable');
	return response.json();
}
