import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env as privateEnv } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import { getStaffCalendarSnapshot } from '$lib/server/staffCalendarSnapshot';
import { requireStaffCalendarAccess } from '$lib/server/staffCalendarAccess';
import {
	buildStaffShiftCalendar,
	resolveStaffShiftName,
	staffShiftFilename
} from '$lib/staffShiftExport';

const DEFAULT_EXPORT_DAYS = 183;
const MAX_EXPORT_DAYS = 370;

function startOfToday(): Date {
	const today = new Date();
	today.setHours(0, 0, 0, 0);
	return today;
}

function exportRange(url: URL): { start: Date; end: Date } {
	const defaultStart = startOfToday();
	const defaultEnd = new Date(defaultStart);
	defaultEnd.setDate(defaultEnd.getDate() + DEFAULT_EXPORT_DAYS);

	const start = url.searchParams.has('start')
		? new Date(url.searchParams.get('start') ?? '')
		: defaultStart;
	const end = url.searchParams.has('end')
		? new Date(url.searchParams.get('end') ?? '')
		: defaultEnd;

	if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()) || end <= start) {
		error(400, 'Choose a valid shift-export date range.');
	}
	if (end.getTime() - start.getTime() > MAX_EXPORT_DAYS * 24 * 60 * 60 * 1000) {
		error(400, `Shift exports are limited to ${MAX_EXPORT_DAYS} days.`);
	}
	return { start, end };
}

export const GET: RequestHandler = async ({ fetch, url, cookies }) => {
	const user = await requireStaffCalendarAccess({
		token: cookies.get('token'),
		leashURL: publicEnv.PUBLIC_LEASH_ENDPOINT,
		fetch
	});
	const staffName = resolveStaffShiftName(user);
	if (!staffName) {
		error(409, 'Your account is not yet matched to a staff-calendar name.');
	}

	const endpoint = privateEnv.STAFF_CALENDAR_ENDPOINT;
	if (!endpoint) error(500, 'No calendar endpoint configured.');
	const { start, end } = exportRange(url);

	let events;
	try {
		events = await getStaffCalendarSnapshot(endpoint).getEventsBetween(start, end);
	} catch {
		console.error('Failed to read staff calendar data for shift export.');
		error(500, 'Unable to create the shift export.');
	}

	const calendar = buildStaffShiftCalendar(staffName, events);
	return new Response(calendar, {
		headers: {
			'Content-Type': 'text/calendar; charset=utf-8',
			'Content-Disposition': `attachment; filename="${staffShiftFilename(staffName)}"`,
			'Cache-Control': 'private, no-store'
		}
	});
};
