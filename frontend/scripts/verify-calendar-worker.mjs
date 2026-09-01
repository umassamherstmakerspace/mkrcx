import { createServer } from 'node:http';
import { resolve } from 'node:path';
import { Worker } from 'node:worker_threads';

const fixture = [
	'BEGIN:VCALENDAR',
	'VERSION:2.0',
	'PRODID:-//Makerspace//Worker smoke test//EN',
	'BEGIN:VEVENT',
	'UID:worker-smoke-test@example.com',
	'DTSTAMP:20260901T120000Z',
	'DTSTART:20260908T140000Z',
	'DTEND:20260908T150000Z',
	'SUMMARY:Worker smoke test',
	'SEQUENCE:0',
	'END:VEVENT',
	'END:VCALENDAR',
	''
].join('\r\n');

const server = createServer((_request, response) => {
	response.writeHead(200, { 'Content-Type': 'text/calendar' });
	response.end(fixture);
});

await new Promise((resolveListen) => server.listen(0, '127.0.0.1', resolveListen));
const address = server.address();
if (!address || typeof address === 'string')
	throw new Error('Worker smoke server failed to start.');

const worker = new Worker(resolve('build/workers/staff-calendar.mjs'), {
	workerData: { endpoint: `http://127.0.0.1:${address.port}/calendar.ics` }
});

try {
	await new Promise((resolveWorker, rejectWorker) => {
		const timeout = setTimeout(
			() => rejectWorker(new Error('Calendar worker smoke test timed out.')),
			10_000
		);
		const resolveOnce = () => {
			clearTimeout(timeout);
			resolveWorker();
		};
		const rejectOnce = (error) => {
			clearTimeout(timeout);
			rejectWorker(error);
		};

		worker.on('error', rejectOnce);
		worker.on('message', (message) => {
			if (message.type === 'startup-error' || message.type === 'request-error') {
				rejectOnce(new Error('Calendar worker smoke test failed.'));
				return;
			}
			if (message.type === 'ready') {
				worker.postMessage({
					type: 'between',
					id: 1,
					start: Date.parse('2026-09-08T00:00:00Z'),
					end: Date.parse('2026-09-09T00:00:00Z')
				});
				return;
			}
			if (message.type === 'result') {
				if (message.events.length !== 1 || message.events[0].title !== 'Worker smoke test') {
					rejectOnce(new Error('Calendar worker returned unexpected events.'));
					return;
				}
				resolveOnce();
			}
		});
	});
} finally {
	await worker.terminate();
	await new Promise((resolveClose) => server.close(resolveClose));
}

console.log('Calendar worker smoke test passed.');
