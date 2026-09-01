import { spawn } from 'node:child_process';
import { createServer } from 'node:http';

const fixture = [
	'BEGIN:VCALENDAR',
	'VERSION:2.0',
	'PRODID:-//Makerspace//Prewarm smoke test//EN',
	'BEGIN:VEVENT',
	'UID:prewarm-smoke-test@example.com',
	'DTSTAMP:20260901T120000Z',
	'DTSTART:20260908T140000Z',
	'DTEND:20260908T150000Z',
	'SUMMARY:Niall',
	'SEQUENCE:0',
	'END:VEVENT',
	'END:VCALENDAR',
	''
].join('\r\n');

const staffUser = {
	ID: 7,
	CreatedAt: '2026-01-01T00:00:00Z',
	UpdatedAt: '2026-01-01T00:00:00Z',
	Email: 'person@example.edu',
	CardID: '',
	Name: 'Person',
	Pronouns: '',
	Role: 'staff',
	Type: 'employee',
	GraduationYear: 0,
	Major: '',
	Department: '',
	JobTitle: '',
	Permissions: []
};

let calendarReads = 0;
const fixtureServer = createServer((request, response) => {
	if (request.url === '/calendar.ics') {
		calendarReads += 1;
		response.writeHead(200, { 'Content-Type': 'text/calendar' });
		response.end(fixture);
		return;
	}
	if (request.url?.startsWith('/api/users/self')) {
		response.writeHead(200, { 'Content-Type': 'application/json' });
		response.end(JSON.stringify(staffUser));
		return;
	}
	response.writeHead(404);
	response.end();
});

await new Promise((resolveListen) => fixtureServer.listen(0, '127.0.0.1', resolveListen));
const fixtureAddress = fixtureServer.address();
if (!fixtureAddress || typeof fixtureAddress === 'string') {
	throw new Error('Prewarm fixture server failed to start.');
}

const portProbe = createServer();
await new Promise((resolveListen) => portProbe.listen(0, '127.0.0.1', resolveListen));
const appAddress = portProbe.address();
if (!appAddress || typeof appAddress === 'string')
	throw new Error('Could not reserve an app port.');
const appPort = appAddress.port;
await new Promise((resolveClose) => portProbe.close(resolveClose));

const fixtureOrigin = `http://127.0.0.1:${fixtureAddress.port}`;
const appOrigin = `http://127.0.0.1:${appPort}`;
const app = spawn(process.execPath, ['build'], {
	env: {
		...process.env,
		HOST: '127.0.0.1',
		PORT: String(appPort),
		ORIGIN: appOrigin,
		PUBLIC_LEASH_ENDPOINT: fixtureOrigin,
		STAFF_CALENDAR_ENDPOINT: `${fixtureOrigin}/calendar.ics`
	},
	stdio: ['ignore', 'pipe', 'pipe']
});

let appOutput = '';
app.stdout.on('data', (chunk) => (appOutput += chunk.toString()));
app.stderr.on('data', (chunk) => (appOutput += chunk.toString()));

const delay = (milliseconds) =>
	new Promise((resolveDelay) => setTimeout(resolveDelay, milliseconds));
const deadline = Date.now() + 30_000;

try {
	let homepage;
	while (Date.now() < deadline) {
		try {
			homepage = await fetch(appOrigin);
			break;
		} catch {
			if (app.exitCode !== null) throw new Error(`App exited during prewarm: ${appOutput}`);
			await delay(100);
		}
	}
	if (!homepage?.ok) throw new Error(`App did not become ready during prewarm: ${appOutput}`);

	const calendarURL = `${appOrigin}/staff/calendar?start=2026-09-08&end=2026-09-09`;
	for (let attempt = 0; attempt < 2; attempt += 1) {
		const started = Date.now();
		const response = await fetch(calendarURL, { headers: { Cookie: 'token=staff-token' } });
		const events = await response.json();
		if (!response.ok || events.length !== 1 || events[0].title !== 'Niall') {
			throw new Error('Prewarmed calendar endpoint returned unexpected data.');
		}
		if (Date.now() - started > 2_000) {
			throw new Error('Prewarmed calendar endpoint was unexpectedly slow.');
		}
	}
	if (calendarReads !== 1) {
		throw new Error(`Expected one prewarm source read, received ${calendarReads}.`);
	}
} finally {
	app.kill();
	await new Promise((resolveClose) => fixtureServer.close(resolveClose));
}

console.log('Calendar prewarm smoke test passed.');
