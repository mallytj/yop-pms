import { describe, it, expect, afterEach, beforeEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte/svelte5';
import type { Snippet } from 'svelte';
import AppLayout from './+layout.svelte';
import { shellStore } from '$lib/stores/shell.svelte';

const children = (() => document.createTextNode('')) as unknown as Snippet;

describe('App root layout', () => {
	beforeEach(() => {
		shellStore.topLeft = 'The Ollerod';
		render(AppLayout, { props: { children } });
	});

	afterEach(cleanup);

	it('renders Tape Chart in sidebar', () => {
		expect(screen.getByText('Tape Chart')).toBeTruthy();
	});

	it('renders Housekeeping in sidebar', () => {
		expect(screen.getByText('Housekeeping')).toBeTruthy();
	});

	it('does not render TopBar tabs', () => {
		expect(document.querySelector('.role-tab')).toBeNull();
	});

	it('renders a content area', () => {
		expect(document.querySelector('.body')).toBeTruthy();
	});
});
