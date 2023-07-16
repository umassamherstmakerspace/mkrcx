import { writable } from "svelte/store";
import { DEFAULT_DATE_FORMAT } from "./locale";

export const date_format = writable(DEFAULT_DATE_FORMAT);