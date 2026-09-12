import { describe, it, expect, vi } from 'vitest';
import { asISO8601Date } from '$helpers/dates.js';

const get = vi.fn();
vi.mock('$lib/api/client.js', () => ({
	api: { GET: (...args: unknown[]) => get(...args) }
}));

const { fetchTapeChart } = await import('./tape-chart.js');

const FROM = asISO8601Date('2026-08-20');
const TO = asISO8601Date('2026-10-09');

describe('fetchTapeChart', () => {
	it('requests the tape chart for the given range and include mode', async () => {
		get.mockResolvedValueOnce({ data: { data: { from: '2026-08-20', to: '2026-10-09' } } });
		await fetchTapeChart(FROM, TO, 'shallow');
		expect(get).toHaveBeenCalledWith('/v1/tape-chart', {
			params: { query: { from: '2026-08-20', to: '2026-10-09', include: 'shallow' } },
			fetch: undefined
		});
	});

	it('defaults to a full include when none is given', async () => {
		get.mockResolvedValueOnce({ data: { data: { from: '2026-08-20', to: '2026-10-09' } } });
		await fetchTapeChart(FROM, TO);
		expect(get).toHaveBeenCalledWith(
			'/v1/tape-chart',
			expect.objectContaining({ params: { query: expect.objectContaining({ include: 'full' }) } })
		);
	});

	it('returns the response body on success', async () => {
		const body = { data: { from: '2026-08-20', to: '2026-10-09' } };
		get.mockResolvedValueOnce({ data: body });
		const result = await fetchTapeChart(FROM, TO);
		expect(result).toEqual(body);
	});

	it('throws the API error message when the request fails', async () => {
		get.mockResolvedValueOnce({ error: { message: 'property not found' } });
		await expect(fetchTapeChart(FROM, TO)).rejects.toThrow('property not found');
	});

	it('throws a fallback message when the response has no data and no error message', async () => {
		get.mockResolvedValueOnce({ data: undefined, error: undefined });
		await expect(fetchTapeChart(FROM, TO)).rejects.toThrow('Failed to fetch tape chart');
	});

	it('throws when the response body has no inner data payload', async () => {
		get.mockResolvedValueOnce({ data: { data: undefined } });
		await expect(fetchTapeChart(FROM, TO)).rejects.toThrow('Failed to fetch tape chart');
	});
});
