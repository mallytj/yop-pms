import type { Component } from 'svelte';

export interface TabDef {
	id: string;
	label: string;
	icon: Component;
}

// Global reactive state for the app top bar.
// TapeChart layout mutates properties; TopBar component reads them.
// NOTE: must mutate properties, not reassign `topBar` itself
// (Svelte 5 forbids export of reassignable $state from modules).
export const topBar = $state({
	tabs: [] as TabDef[],
	active: '',
	onchange: undefined as ((id: string) => void) | undefined,
	reset() {
		this.tabs = [];
		this.active = '';
		this.onchange = undefined;
	}
});
