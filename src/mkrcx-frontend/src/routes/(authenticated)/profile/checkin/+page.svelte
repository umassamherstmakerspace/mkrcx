<script lang="ts">
    import QRCode from 'qrcode'
	import type { PageData } from './$types';
    import { Heading, P } from 'flowbite-svelte';

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

<div class="flex flex-col items-center space-y-4 w-full h-full">
    <Heading level={1} class="text-center">Check In</Heading>
    <P class="text-xl">Scan the QR code below to check in.</P>
    <div class="flex-1 flex items-center justify-center aspect-square max-w-full max-h-full"> 
        <img class="object-contain flex-1 h-full w-full" src={imageURI} alt="Check In QR Code" />
    </div>
</div>