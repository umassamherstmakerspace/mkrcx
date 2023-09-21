import Cookies from 'js-cookie';
import { User, type LeashUser, type LeashTraining, type LeashUserUpdate, type LeashUserUpdateRequest, type LeashSelfUpdateRequest } from './types';
import { dev } from '$app/environment';

const LEASH_ENDPOINT = dev ? 'http://127.0.0.1:8000' : '';

export const key = Symbol();

interface LeashSearchUsersResponse {
	count: number;
	users: LeashUser[];
}

interface SearchUsersResponse {
	count: number;
	users: User[];
}

async function leashFetch<T extends object>(
	endpoint: string,
	method: string,
	body?: object,
	noReponse = false
): Promise<T> {
	const r = await fetch(`${LEASH_ENDPOINT}${endpoint}`, {
		method: method,
		headers: {
			Authorization: `Bearer ${Cookies.get('token')}`,
			'Content-Type': 'application/json'
		},
		redirect: 'follow',
		mode: 'cors',
		cache: 'no-cache',
		credentials: 'same-origin',
		body: JSON.stringify(body)
	});
	return await (r.status !== 200
		? Promise.reject(new Error(await r.text()))
		: noReponse
		? r.text()
		: r.json());
}

export async function searchUsers(
	query: string,
	limit = 30,
	offset = 0,
	only_enabled = false,
	with_training = true,
	with_updates = true
): Promise<SearchUsersResponse> {
	const result = await leashFetch<LeashSearchUsersResponse>(
		`/api/users/search?q=${query}&allow_empty_body=true&only_enabled=${only_enabled}&with_trainings=${with_training}&with_updates=${with_updates}&limit=${limit}&offset=${offset}`,
		'GET'
	);

	return {
		count: result.count,
		users: result.users.map((user) => new User(user))
	};
}

export async function getUserByEmail(
	email: string,
	with_training = true,
	with_updates = true
): Promise<User> {
	const result = await leashFetch<LeashUser>(
		`/api/users?email=${email}&with_trainings=${with_training}&with_updates=${with_updates}`,
		'GET'
	);

	return new User(result);
}

export async function getUserById(
	id: number,
	with_training = true,
	with_updates = true
): Promise<User> {
	const result = await leashFetch<LeashUser>(
		`/api/users?id=${id}&with_trainings=${with_training}&with_updates=${with_updates}`,
		'GET'
	);

	return new User(result);
}

export async function updateUser(
	email: string,
	update: LeashUserUpdateRequest
): Promise<void> {
	await leashFetch(
		`/api/users`,
		'PUT',
		{
			email,
			...update
		},
		true
	);
}

export async function getSelf(with_training = true): Promise<User> {
	const result = await leashFetch<LeashUser>(
		`/api/users/self?with_trainings=${with_training}`,
		'GET'
	);

	return new User(result);
}

export async function updateSelf(
	update: LeashSelfUpdateRequest
): Promise<void> {
	await leashFetch(
		`/api/users/self`,
		'PUT',
		{
			...update
		},
		true
	);
}

export async function getTrainings(email: string): Promise<LeashTraining[]> {
	const result = await leashFetch<LeashTraining[]>(
		`/api/training?email=${email}&include_deleted=true`,
		'GET'
	);
	return result;
}

export async function createTraining(email: string, training_type: string): Promise<void> {
	await leashFetch(
		`/api/training`,
		'POST',
		{
			email,
			training_type
		},
		true
	);
}

export async function removeTraining(email: string, training_type: string): Promise<void> {
	await leashFetch(
		`/api/training`,
		'DELETE',
		{
			email,
			training_type
		},
		true
	);
}

export async function getUserUpdates(email: string): Promise<LeashUserUpdate[]> {
	const result = await leashFetch<LeashUserUpdate[]>(`/api/updates?email=${email}`, 'GET');
	return result;
}

interface LeashTokenRefresh {
	token: string;
	expires_at: string;
}

export async function refreshTokens(): Promise<boolean> {
	try {
		const refresh = await leashFetch<LeashTokenRefresh>(`/auth/refresh`, 'GET');
		Cookies.set('token', refresh.token, {
			expires: new Date(refresh.expires_at),
			sameSite: 'strict'
		});
		return true;
	} catch (e) {
		return false;
	}
}

export async function validateToken(): Promise<boolean> {
	try {
		await leashFetch(`/auth/validate`, 'GET', undefined, true);
		return true;
	} catch (e) {
		console.log(e);
		return false;
	}
}

export async function login(return_to: string): Promise<void> {
	window.location.href = `${LEASH_ENDPOINT}/auth/login?return=${return_to}`;
}
