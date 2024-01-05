<script lang="ts">
    import { afterNavigate } from '$app/navigation';
    import { base } from '$app/paths'
	import { login } from '$lib/leash';
    import { page } from '$app/stores';
    import Cookies from 'js-cookie';

    let previousPage : string = base;

    afterNavigate(({from}) => {
        previousPage = from?.url?.href || previousPage;
        console.log(previousPage);

        let token = $page.url.searchParams.get('token');
        let state = $page.url.searchParams.get('state');
        let expires_at = $page.url.searchParams.get('expires_at');


        if (token && state && expires_at) {
            Cookies.set('token', token, {
                expires: new Date(expires_at),
                sameSite: 'strict'
            });

            const ret = atob(state);

            window.location.href = ret;
        } else {
            login($page.url.href || '', previousPage);
        }
    }) 
</script>