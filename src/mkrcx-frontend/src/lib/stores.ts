import { derived, writable, type Readable, type Writable } from "svelte/store";
import type { User } from "./leash";

export type Themes = "light" | "dark" | "system";

export const theme: Writable<Themes> = writable("system");
export const isDark: Readable<boolean> = derived(theme, ($theme) => {
    switch ($theme) {
        case "light":
            return false;
        case "dark":
            return true;
        case "system":
            return window.matchMedia("(prefers-color-scheme: dark)").matches;
        default:
            return false;
    }
});

export const user: Writable<User | null> = writable(null);

export type DateTimeFormats = "ISO" | "US" | "EU";
export const dateLocale: Writable<DateTimeFormats> = writable("ISO");

export const dateFormat: Readable<string> = derived(dateLocale, ($dateLocale) => {
    switch ($dateLocale) {
        case "US":
            return "MM/dd/yyyy";
        case "EU":
            return "dd/MM/yyyy";
        case "ISO":
        default:
            return "yyyy-MM-dd";
            
    }
});

export const timeFormat: Readable<string> = derived(dateLocale, ($dateLocale) => {
    switch ($dateLocale) {
        case "US":
            return "hh:mm:ss a";
        case "EU":
            return "HH:mm:ss";
        case "ISO":
        default:
            // return "HH:mm:ss.SSSXXX";
            return "HH:mm:ssXXX";
            
    }
});


export const dateTimeJoiner: Readable<string> = derived(dateLocale, ($dateLocale) => {
    switch ($dateLocale) {
        case "US":
            return " ";
        case "EU":
        case "ISO":
        default:
            return "'T'";
            
    }
});

export const dateTimeFormat: Readable<string> = derived([dateFormat, timeFormat, dateTimeJoiner], ([$dateFormat, $timeFormat, $dateTimeJoiner]) => {
    return `${$dateFormat}${$dateTimeJoiner}${$timeFormat}`;
});