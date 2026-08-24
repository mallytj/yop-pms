import { describe, it, expect } from 'vitest';
import { goWithParams } from './url.js';

describe('goWithParams', () => {
	it('sets a new query param while keeping existing ones', () => {
		const url = new URL('http://localhost/tape-chart?from=2026-08-01');

		const result = goWithParams(url, { to: '2026-08-10' });

		expect(result).toBe('?from=2026-08-01&to=2026-08-10');
	});

	it('overwrites an existing query param with the same key', () => {
		const url = new URL('http://localhost/tape-chart?from=2026-08-01');

		const result = goWithParams(url, { from: '2026-09-01' });

		expect(result).toBe('?from=2026-09-01');
	});

	it('removes a query param when given a falsy value', () => {
		const url = new URL('http://localhost/tape-chart?from=2026-08-01&to=2026-08-10');

		const result = goWithParams(url, { to: '' });

		expect(result).toBe('?from=2026-08-01');
	});

	it('returns the bare pathname when no query params remain', () => {
		const url = new URL('http://localhost/tape-chart?from=2026-08-01');

		const result = goWithParams(url, { from: '' });

		expect(result).toBe('/tape-chart');
	});
});
