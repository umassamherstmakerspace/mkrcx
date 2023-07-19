import { writable } from "svelte/store";
import { DEFAULT_DATE_FORMAT, DEFAULT_THEME } from "./defaults";

export const date_format = writable(DEFAULT_DATE_FORMAT);
export const theme = writable(DEFAULT_THEME);