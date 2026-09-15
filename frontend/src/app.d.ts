// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
	const __APP_VERSION__: string;
	const __BUILD_TIME__: string;

	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
