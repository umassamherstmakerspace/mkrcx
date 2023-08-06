import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import Icons from 'unplugin-icons/vite';
import { FileSystemIconLoader } from 'unplugin-icons/loaders';
import { ViteFaviconsPlugin } from 'vite-plugin-favicon2';

export default defineConfig({
	plugins: [
		sveltekit(),
		Icons({
			compiler: 'svelte',
			customCollections: {
				custom: FileSystemIconLoader('assets/iconfont', (svg) =>
					svg.replace(/^<svg /, '<svg fill="currentColor" ')
				)
			}
		}),
		ViteFaviconsPlugin({
			logo: './static/favicon.png'
		})
	]
});
