import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	const propertyId = env.VITE_DEV_PROPERTY_ID || '';

	return {
		plugins: [sveltekit()],
		server: {
			proxy: {
				'/v1': {
					target: 'http://localhost:8080',
					changeOrigin: true,
					configure: (proxy) => {
						proxy.on('proxyReq', (proxyReq) => {
							if (propertyId) {
								proxyReq.setHeader('X-Property-ID', propertyId);
							}
						});
					}
				},
				'/swagger': {
					target: 'http://localhost:8080',
					changeOrigin: true
				}
			}
		}
	};
});
