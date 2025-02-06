<script lang="ts">
	import QRCode from 'qrcode';
	import type { PageData } from './$types';
	import { Heading } from 'flowbite-svelte';

	export let data: PageData;
	let { user } = data;

	let imageURI = '';

	async function loadQRCode() {
		if (imageURI) {
			URL.revokeObjectURL(imageURI);
		}

		const res = await user.checkin();

		const qrSVG = await QRCode.toString(res.token, { errorCorrectionLevel: 'H', type: 'svg' });
		const blob = new Blob([qrSVG], { type: 'image/svg+xml' });
		imageURI = URL.createObjectURL(blob);

		setTimeout(loadQRCode, 60 * 1000);
	}

	loadQRCode();
</script>

<div class="flex h-full w-full flex-col items-center space-y-4">
	<Heading level={1} class="text-center">QR Code</Heading>
	<div class="flex aspect-square max-h-full max-w-full flex-1 items-center justify-center">
		<img class="h-full w-full flex-1 object-contain" src={imageURI} alt="Check In QR Code" />
	</div>
</div>
