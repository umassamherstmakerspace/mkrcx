import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { shipsLogAccountChooserUrl } from '$lib/shipsLog';

export const GET: RequestHandler = () => redirect(302, shipsLogAccountChooserUrl());
