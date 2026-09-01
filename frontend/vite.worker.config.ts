import { defineConfig } from 'vite';

export default defineConfig({
	build: {
		ssr: 'src/lib/server/staffCalendarWorker.ts',
		target: 'node22',
		outDir: 'build/workers',
		emptyOutDir: false,
		rollupOptions: {
			output: {
				entryFileNames: 'staff-calendar.mjs'
			}
		}
	}
});
