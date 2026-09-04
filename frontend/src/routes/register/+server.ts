import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { registrationAccountChooserUrl } from '$lib/registration';

export const GET: RequestHandler = () => redirect(302, registrationAccountChooserUrl());
