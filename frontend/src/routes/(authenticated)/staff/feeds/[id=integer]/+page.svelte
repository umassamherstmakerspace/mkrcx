<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { applyFeedRetention, FeedRecoveryBuffer, formatFeedTimestamp } from '$lib/feedRecovery';
	import { LeashAPIError } from '$lib/leash';
	import type { LeashFeed, LeashFeedItem, LeashFeedSocketEvent } from '$lib/leash';
	import type { PageData } from './$types';

	export let data: PageData;

	type ConnectionState = 'connecting' | 'live' | 'recovering' | 'offline';

	let feed: LeashFeed | undefined;
	let items: LeashFeedItem[] = [];
	let state: ConnectionState = 'connecting';
	let failure = '';
	let socket: WebSocket | undefined;
	let socketReady = false;
	let mounted = false;
	let destroyed = false;
	let blocked = false;
	let activeFeedID = 0;
	let generation = 0;
	let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
	let pollTimer: ReturnType<typeof setInterval> | undefined;
	let sessionReloadTimer: ReturnType<typeof setTimeout> | undefined;
	let retryDelay = 1000;
	let recoveryComplete = false;
	let recovery = new FeedRecoveryBuffer();

	$: feedID = Number($page.params.id);
	$: if (mounted && feedID !== activeFeedID) startFeed(feedID);

	function mergeItems(incoming: LeashFeedItem[]) {
		items = applyFeedRetention(recovery.mergeSocket(items, incoming), feed?.Name);
	}

	function itemTitle(item: LeashFeedItem) {
		return item.UserDisplayName ?? item.Title;
	}

	function feedTitle() {
		if (feed?.Name === 'signin') return 'Front Desk Check-in';
		return feed?.Name?.replaceAll('-', ' ') ?? 'Live feed';
	}

	function clearConnection() {
		const currentSocket = socket;
		socket = undefined;
		socketReady = false;
		currentSocket?.close();
		if (reconnectTimer) clearTimeout(reconnectTimer);
		if (pollTimer) clearInterval(pollTimer);
		reconnectTimer = undefined;
		pollTimer = undefined;
	}

	function handleAccessFailure(error: unknown): boolean {
		if (!(error instanceof LeashAPIError) || (error.status !== 401 && error.status !== 403)) {
			return false;
		}
		generation += 1;
		blocked = true;
		items = [];
		recoveryComplete = false;
		state = 'offline';
		clearConnection();
		if (error.status === 401) {
			window.location.assign(`/login?return_to=${encodeURIComponent($page.url.pathname)}`);
		} else {
			failure =
				'Feed access was denied or revoked. Previously displayed activity has been cleared.';
		}
		return true;
	}

	async function catchUp(id: number, run: number, refreshLatest = false): Promise<boolean> {
		try {
			const initialLoad = recovery.cursor === 0;
			if (refreshLatest && !initialLoad) {
				const latest = await data.api.feedItems(id, { limit: 100 }, true);
				if (destroyed || run !== generation) return false;
				items = applyFeedRetention(recovery.mergeHTTP(items, latest), feed?.Name);
			}
			let incoming: LeashFeedItem[];
			do {
				incoming = await data.api.feedItems(
					id,
					initialLoad ? { limit: 100 } : { afterId: recovery.cursor, limit: 100 },
					true
				);
				if (destroyed || run !== generation) return false;
				items = applyFeedRetention(recovery.mergeHTTP(items, incoming), feed?.Name);
			} while (!initialLoad && incoming.length === 100);
			recoveryComplete = true;
			failure = '';
			return true;
		} catch (error) {
			if (destroyed || run !== generation) return false;
			if (handleAccessFailure(error)) return false;
			recoveryComplete = false;
			state = 'offline';
			failure = error instanceof Error ? error.message : 'Unable to catch up.';
			return false;
		}
	}

	function scheduleRetry(run: number, retry: () => void) {
		if (destroyed || blocked || run !== generation) return;
		if (reconnectTimer) clearTimeout(reconnectTimer);
		reconnectTimer = setTimeout(retry, retryDelay);
		retryDelay = Math.min(retryDelay * 2, 15000);
		state = 'recovering';
	}

	function connect(id: number, run: number) {
		if (destroyed || blocked || run !== generation) return;
		recoveryComplete = false;
		socketReady = false;
		state = recovery.cursor > 0 ? 'recovering' : 'connecting';
		const currentSocket = data.api.openFeedSocket(id);
		socket = currentSocket;
		const readyTimer = setTimeout(() => currentSocket.close(), 10000);
		currentSocket.addEventListener('message', async (message) => {
			if (destroyed || run !== generation || socket !== currentSocket) return;
			try {
				const event = JSON.parse(String(message.data)) as LeashFeedSocketEvent;
				if (event.type === 'feed.ready' && event.feed_id === id) {
					clearTimeout(readyTimer);
					socketReady = true;
					retryDelay = 1000;
					const recovered = await catchUp(id, run);
					if (
						recovered &&
						socket === currentSocket &&
						currentSocket.readyState === WebSocket.OPEN
					) {
						state = 'live';
					}
				} else if (event.type === 'feed_item.created' && event.feed_id === id) {
					mergeItems([event.item]);
					if (socketReady && recoveryComplete) state = 'live';
				}
			} catch {
				// Ignore unknown protocol messages; cursor recovery remains authoritative.
			}
		});
		currentSocket.addEventListener('close', async () => {
			clearTimeout(readyTimer);
			if (destroyed || run !== generation || socket !== currentSocket) return;
			socketReady = false;
			socket = undefined;
			await catchUp(id, run);
			scheduleRetry(run, () => connect(id, run));
		});
		currentSocket.addEventListener('error', () => currentSocket.close());
	}

	async function loadFeed(id: number, run: number) {
		try {
			const loadedFeed = await data.api.feedFromID(id, true);
			if (destroyed || run !== generation) return;
			feed = loadedFeed;
			await catchUp(id, run);
			if (blocked || destroyed || run !== generation) return;
			connect(id, run);
			pollTimer = setInterval(async () => {
				const recovered = await catchUp(id, run, true);
				if (recovered && socketReady && socket?.readyState === WebSocket.OPEN) state = 'live';
			}, 5000);
		} catch (error) {
			if (destroyed || run !== generation) return;
			if (handleAccessFailure(error)) return;
			state = 'offline';
			failure = error instanceof Error ? error.message : 'Unable to open this feed.';
			scheduleRetry(run, () => void loadFeed(id, run));
		}
	}

	function startFeed(id: number) {
		generation += 1;
		const run = generation;
		clearConnection();
		activeFeedID = id;
		blocked = false;
		feed = undefined;
		items = [];
		failure = '';
		state = 'connecting';
		retryDelay = 1000;
		recoveryComplete = false;
		recovery = new FeedRecoveryBuffer();
		void loadFeed(id, run);
	}

	function itemTone(level: number) {
		if (level >= 4) return 'border-red-400 bg-red-50 dark:border-red-800 dark:bg-red-950';
		if (level >= 2) return 'border-amber-400 bg-amber-50 dark:border-amber-800 dark:bg-amber-950';
		return 'border-emerald-400 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950';
	}

	function refreshSessionWhenHealthy() {
		if (destroyed) return;
		if (state === 'live') {
			window.location.reload();
			return;
		}
		sessionReloadTimer = setTimeout(refreshSessionWhenHealthy, 5 * 60 * 1000);
	}

	onMount(() => {
		mounted = true;
		startFeed(feedID);
		// A full reload exercises the server-side session refresh and replaces the
		// browser cookie before a permanently open HUD can reach token expiry.
		sessionReloadTimer = setTimeout(refreshSessionWhenHealthy, 6 * 60 * 60 * 1000);

		return () => {
			destroyed = true;
			clearConnection();
			if (sessionReloadTimer) clearTimeout(sessionReloadTimer);
		};
	});
</script>

<svelte:head><title>{feedTitle()} HUD · mkr.cx</title></svelte:head>

<main class="mx-auto flex min-h-full max-w-6xl flex-col gap-6" aria-live="polite">
	<header
		class="sticky top-0 z-10 -mx-2 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-gray-200 bg-white/95 p-5 shadow-sm backdrop-blur dark:border-gray-700 dark:bg-gray-950/95"
	>
		<div>
			<a
				href="/staff/feeds"
				class="text-sm font-medium text-blue-600 hover:underline dark:text-blue-400">← All feeds</a
			>
			<h1 class="mt-1 text-3xl font-bold capitalize text-gray-950 dark:text-white">
				{feedTitle()}
			</h1>
		</div>
		<div
			class="flex items-center gap-3 rounded-full border border-gray-200 px-4 py-2 dark:border-gray-700"
		>
			<span
				class:animate-pulse={state !== 'offline'}
				class="h-3 w-3 rounded-full"
				class:bg-emerald-500={state === 'live'}
				class:bg-amber-500={state === 'connecting' || state === 'recovering'}
				class:bg-red-500={state === 'offline'}
			></span>
			<span class="font-semibold capitalize text-gray-800 dark:text-gray-100">{state}</span>
		</div>
	</header>

	{#if failure}
		<div
			class="rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100"
		>
			{failure}{#if !blocked}
				Retrying automatically.{/if}
		</div>
	{/if}

	{#if items.length === 0}
		<div
			class="flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-gray-300 p-8 text-center dark:border-gray-700"
		>
			<p class="text-2xl font-semibold text-gray-800 dark:text-gray-100">
				Waiting for the next card tap
			</p>
			<p class="mt-2 text-gray-500 dark:text-gray-400">
				New activity will appear here automatically.
			</p>
		</div>
	{:else}
		<section class="grid gap-4" aria-label="Recent feed activity">
			{#each items as item, index (item.ID)}
				<article
					class={`rounded-2xl border-l-8 p-5 shadow-sm transition ${itemTone(item.LogLevel)} ${index === 0 ? 'ring-2 ring-blue-400/50' : ''}`}
				>
					<div class="flex flex-wrap items-start justify-between gap-3">
						<div>
							<p class="text-2xl font-bold text-gray-950 dark:text-white">{itemTitle(item)}</p>
							{#if item.UserEmail}
								<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{item.UserEmail}</p>
							{/if}
							<p class="mt-2 text-lg text-gray-700 dark:text-gray-200">{item.Message}</p>
						</div>
						<time
							class="whitespace-nowrap font-mono text-sm text-gray-500 dark:text-gray-400"
							datetime={item.CreatedAt}
						>
							{formatFeedTimestamp(item.CreatedAt)}
						</time>
					</div>
				</article>
			{/each}
		</section>
	{/if}
</main>
