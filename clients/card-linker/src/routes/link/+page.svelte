<script lang="ts">
	import QrScanner from 'qr-scanner';
	import { Mutex } from 'async-mutex';
	import { onMount, onDestroy } from 'svelte';
	import { invoke } from '@tauri-apps/api/tauri'
	import { goto } from '$app/navigation';

	onMount(async () => {
        let running = true;
        await invoke('set_reader');

        let name = await invoke('get_user');
        console.log(name);

        while (running) {
            try {
                const cardNumber = await invoke('get_card_id');
                await invoke('set_card', { cardNumber });
                await goto("/");
                running = false;
            } catch (e) {
                console.error(e);
            }
        }
    });
</script>

<div class="container h-full mx-auto flex justify-center items-center">
	<div id="qr-overlay" />
	<video id="qr-video" class="w-full -z-10">
		<track kind="captions" />
	</video>
</div>
