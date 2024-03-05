<script lang="ts">
	import { afterNavigate, goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { page } from '$app/stores';
	import Cookies from 'js-cookie';
	import type { PageData } from './$types';

	export let data: PageData;
	const { user, api } = data;

	let previousPage: string = base;

	afterNavigate(({ from }) => {
		previousPage = from?.url?.href || previousPage;
		if (previousPage.includes('/login')) {
			previousPage = '/';
		} else if (previousPage == '') {
			previousPage = '/';
		}

		let token = $page.url.searchParams.get('token');
		let state = $page.url.searchParams.get('state');
		let expires_at = $page.url.searchParams.get('expires_at');

		if (token && state && expires_at) {
			Cookies.set('token', token, {
				expires: new Date(expires_at),
				sameSite: 'strict'
			});

			let ret = atob(state);

			if (ret.includes('/login')) {
				ret = '/';
			}

			window.location.href = ret;
		} else {
			if (user) {
				window.location.href = previousPage;
			} else {
				window.location.href = api.login($page.url.href || '', previousPage);
			}
		}
	});
</script>
