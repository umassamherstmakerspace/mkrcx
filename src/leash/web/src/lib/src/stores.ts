import { derived, writable, type Writable } from 'svelte/store';
import { DEFAULT_DATE_FORMAT, DEFAULT_THEME } from './defaults';
import type { User } from './types';
import type { Dayjs } from 'dayjs';
import type { Readable } from 'svelte/motion';

export const date_format = writable(DEFAULT_DATE_FORMAT);
export const theme = writable(DEFAULT_THEME);
export const user: Writable<User | null> = writable(null);

export const screenH = writable(900);
export const screenW = writable(900);
export const mobileThreshold = writable(800);
export const mobile = derived(
	[screenW, mobileThreshold],
	([$screenW, $mobileThreshold]) => $screenW < $mobileThreshold
);


export const timestampCreator: (date: Dayjs) => Readable<string> = (date: Dayjs) => {
	return derived(date_format, ($date_format) => date.format($date_format));
};