import { vi } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$app/stores', () => ({
	page: {
		subscribe(fn: Function) {
			fn({ url: { pathname: '/reservations' } });
			return () => {};
		}
	}
}));
