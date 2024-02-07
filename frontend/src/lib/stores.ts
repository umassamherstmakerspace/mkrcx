import { getContext } from 'svelte';
import type { Readable, Writable } from 'svelte/store';

export type Themes = 'light' | 'dark' | 'system';
export type DateTimeFormats = 'ISO' | 'US' | 'EU';

export const getTheme: () => Writable<Themes> = () => getContext('theme');
export const getIsDark: () => Readable<boolean> = () => getContext('isDark');
export const getDateLocale: () => Writable<DateTimeFormats> = () => getContext('dateLocale');
export const getDateFormat: () => Readable<string> = () => getContext('dateFormat');
export const getTimeFormat: () => Readable<string> = () => getContext('timeFormat');
export const getDateTimeJoiner: () => Readable<string> = () => getContext('dateTimeJoiner');
export const getDateTimeFormat: () => Readable<string> = () => getContext('dateTimeFormat');
