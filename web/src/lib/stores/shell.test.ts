import { describe, it, expect, beforeEach } from 'vitest';
import { shellStore } from './shell.svelte';

describe('shellStore', () => {
	beforeEach(() => {
		shellStore.topLeft = '🏨 The Ollerod';
		shellStore.topRight = '';
	});

	it('defaults topLeft to hotel welcome message', () => {
		expect(shellStore.topLeft).toBe('🏨 The Ollerod');
	});

	it('allows setting topLeft', () => {
		shellStore.topLeft = 'Dashboard';
		expect(shellStore.topLeft).toBe('Dashboard');
	});

	it('defaults topRight to empty string', () => {
		expect(shellStore.topRight).toBe('');
	});

	it('allows setting topRight', () => {
		shellStore.topRight = 'Actions';
		expect(shellStore.topRight).toBe('Actions');
	});
});
