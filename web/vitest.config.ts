import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'path';

export default defineConfig({
	plugins: [svelte()],
	resolve: {
		conditions: ['browser'],
		alias: [
			{ find: '$app/navigation', replacement: path.resolve('./__mocks__/$app/navigation.ts') },
			{ find: '$app/stores', replacement: path.resolve('./__mocks__/$app/stores.ts') },
			{ find: '$lib', replacement: path.resolve('./src/lib') },
			{ find: '$components', replacement: path.resolve('./src/lib/components') },
			{ find: '$helpers', replacement: path.resolve('./src/lib/helpers') },
			{ find: '$stores', replacement: path.resolve('./src/lib/stores') },
			{ find: '$types', replacement: path.resolve('./src/lib/types') },
			{ find: '$actions', replacement: path.resolve('./src/lib/actions') }
		]
	},
	test: {
		environment: 'jsdom',
		include: ['src/**/*.test.ts'],
		setupFiles: ['./vitest.setup.ts']
	}
});
