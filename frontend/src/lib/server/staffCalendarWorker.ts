import { parentPort, workerData } from 'node:worker_threads';
import { CalendarSet } from '../calendar';

type WorkerConfig = {
	endpoint: string;
};

type CalendarRequest = {
	type: 'between';
	id: number;
	start: number;
	end: number;
};

if (!parentPort) {
	throw new Error('Staff calendar worker requires a parent port.');
}

const port = parentPort;
const { endpoint } = workerData as WorkerConfig;

try {
	const response = await fetch(endpoint);
	if (!response.ok) {
		throw new Error('Calendar source request failed.');
	}

	const calendar = CalendarSet.cleanAndParse(await response.text());
	port.on('message', (request: CalendarRequest) => {
		if (request.type !== 'between') return;

		try {
			port.postMessage({
				type: 'result',
				id: request.id,
				events: calendar.between(new Date(request.start), new Date(request.end))
			});
		} catch {
			port.postMessage({ type: 'request-error', id: request.id });
		}
	});
	port.postMessage({ type: 'ready' });
} catch {
	port.postMessage({ type: 'startup-error' });
	port.close();
}
