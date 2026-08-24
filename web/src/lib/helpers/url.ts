/**
 * Builds a path+query string from `url` with `params` merged in.
 * A falsy value deletes that key instead of setting it.
 */
export function goWithParams(url: URL, params: Record<string, string>): string {
	const searchParams = new URLSearchParams(url.searchParams);
	for (const [key, value] of Object.entries(params)) {
		if (value) searchParams.set(key, value);
		else searchParams.delete(key);
	}
	const queryString = searchParams.toString();
	return queryString ? `?${queryString}` : url.pathname;
}
