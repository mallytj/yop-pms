import type { ComponentType } from 'svelte';

export interface Tab {
	id: string;
	label: string;
	icon?: ComponentType<any>;
}

interface TopBarState {
	tabs: Tab[];
	active: string;
	onchange: (id: string) => void;
}

const state = $state<TopBarState>({
	tabs: [],
	active: '',
	onchange: () => {}
});

export const topBarStore = {
	get tabs() { return state.tabs; },
	set tabs(value: Tab[]) { state.tabs = value; },
	get active() { return state.active; },
	set active(value: string) { state.active = value; },
	get onchange() { return state.onchange; },
	set onchange(value: (id: string) => void) { state.onchange = value; },
	reset() {
		state.tabs = [];
		state.active = '';
		state.onchange = () => {};
	}
};
