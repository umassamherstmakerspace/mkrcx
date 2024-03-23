export type ConvertFields<T, V> = T extends string | number | boolean | undefined
	? V
	: T extends object
		? { [K in keyof T]: ConvertFields<T[K], V> }
		: never;
