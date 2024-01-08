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
            theme.set("system");
            return false;
    }
});

export const user: Writable<User | null> = writable(null);


export const ISODate = "yyyy-MM-dd'T'HH:mm:ss.SSSXXX";
export const USDate = "MM/dd/yyyy hh:mm:ss a";
export const dateTimeFormat: Writable<string> = writable(ISODate);