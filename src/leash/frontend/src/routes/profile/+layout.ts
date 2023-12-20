import { login, validateToken } from '$lib/src/leash';
import type { LayoutLoad } from './$types';

export const load = (async () => {
	if (!(await validateToken())) {
		login(window.location.href);
	}
}) satisfies LayoutLoad;
