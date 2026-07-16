export const page = {
	subscribe(fn: Function) {
		fn({ url: { pathname: '/reservations' }, params: {}, route: { id: null }, status: 200, error: null, data: {}, form: null, state: {} });
		return () => {};
	}
};
