export function safeLoginReturn(
	candidate: string | null | undefined,
	origin: string,
	fallback: string
): string {
	if (!candidate) return fallback;
	try {
		const expectedOrigin = new URL(origin).origin;
		const target = new URL(candidate, expectedOrigin);
		if (target.origin !== expectedOrigin || target.username || target.password) return fallback;
		return target.toString();
	} catch {
		return fallback;
	}
}
