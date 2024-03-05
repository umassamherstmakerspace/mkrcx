import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { env } from '$env/dynamic/public';

export const load: PageLoad = () => {
    const endpoint = env.PUBLIC_MPSK_ENDPOINT;

    if (!endpoint) {
        throw new Error('MPSK endpoint not set');
    }

    redirect(301, `${endpoint}`);
};