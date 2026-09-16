import { error } from '@sveltejs/kit';
import type { Printer } from '$lib/printers/prototype-data';

export async function readPrinterDetail(
	fetch: typeof globalThis.fetch,
	endpoint: string | undefined,
	token: string | undefined,
	id: string
) {
	if (!token) error(401, 'Staff sign-in required.');
	if (!endpoint) error(503, 'Printer status unavailable.');
	if (!/^[a-z0-9][a-z0-9-]{0,79}$/.test(id)) error(404, 'Printer not found.');
	const options = () => ({
		headers: { Authorization: `Bearer ${token}` },
		signal: AbortSignal.timeout(5000)
	});
	const base = endpoint.replace(/\/$/, '');
	let response: Response;
	try {
		response = await fetch(`${base}/api/printer-fleet/staff`, options());
	} catch {
		error(503, 'Printer status unavailable.');
	}
	if (response.status === 401 || response.status === 403)
		error(response.status, 'Staff access required.');
	if (!response.ok) error(503, 'Printer status unavailable.');
	const fleet = (await response.json()) as {
		audience: string;
		stale: boolean;
		printers: Printer[];
	};
	if (fleet.audience !== 'staff' || !Array.isArray(fleet.printers))
		error(503, 'Printer status unavailable.');
	const printer = fleet.printers.find((printer) => printer.id === id);
	if (!printer) error(404, 'Printer not found.');
	let canManage = false;
	try {
		const permissions = await fetch(`${base}/api/users/self/permissions`, options());
		if (permissions.ok) {
			const list = await permissions.json();
			canManage = Array.isArray(list) && list.includes('leash.printers:manage');
		}
	} catch {
		/* Reading a printer does not require management access. */
	}
	return { printer: { ...printer, stale: fleet.stale || printer.stale }, canManage };
}
