<script lang="ts">
	import QrScanner from 'qr-scanner';
	import { Mutex } from 'async-mutex';
	import { onMount, onDestroy } from 'svelte';
	import { invoke } from '@tauri-apps/api/tauri'
	import { goto } from '$app/navigation';

	let scanning = false;
	const qrMutex = new Mutex();
	let qrScanner: QrScanner;
	
	onMount(async () => {
		const videoElement = document.getElementById('qr-video') as HTMLVideoElement;
		const divElement = document.getElementById('qr-overlay') as HTMLDivElement;
        await invoke('clear_user');

		qrScanner = new QrScanner(
			videoElement,
			async (result) => {
				if (!scanning) return;
				const release = await qrMutex.acquire();

				try {
					if (!scanning) return;
					scanning = false;

					await invoke('get_checkin', { token: result.data });
                    await goto("/link");
				} catch(e) {
                    console.error("Error: ", e);
                } finally {
                    scanning = true;
					release();
				}
			},
			{
				highlightCodeOutline: true,
				calculateScanRegion: (video) => {
					return { x: 0, y: 0, width: video.width * 0.6, height: video.height * 0.8 };
				},
				overlay: divElement
			}
		);

		const release = await qrMutex.acquire();
		try {
			if (await QrScanner.hasCamera()) {
				const cameras = await QrScanner.listCameras();
				console.log('cameras:', cameras);
				qrScanner.setCamera(cameras[0].id);
				qrScanner.start();
			} else {
				console.error('No camera found');
			}
			scanning = true;
		} finally {
			release();
		}
	})

	onDestroy(() => {
		qrScanner.destroy();
	})
</script>

<div class="container h-full mx-auto flex justify-center items-center">
	<div id="qr-overlay" />
	<video id="qr-video" class="w-full -z-10">
		<track kind="captions" />
	</video>
</div>
