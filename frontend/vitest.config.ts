import { svelte } from "@sveltejs/vite-plugin-svelte";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

/**
 * Отдельная конфигурация Vitest: для юнит-тестов не нужен весь плагин SvelteKit,
 * достаточно компилятора Svelte (файлы `*.svelte.ts` с рунами) и алиаса `$lib`.
 *
 * `conditions: ["browser"]` нужен компонентным тестам: без него Vitest
 * резолвит серверную сборку Svelte, где `mount()` недоступен.
 */
export default defineConfig({
	plugins: [svelte()],
	resolve: {
		conditions: ["browser"],
		alias: {
			$lib: fileURLToPath(new URL("./src/lib", import.meta.url)),
		},
	},
	test: {
		environment: "jsdom",
		include: ["src/**/*.test.ts"],
		restoreMocks: true,
		unstubGlobals: true,
		// Bun читает TZ один раз на старте процесса и игнорирует позднее
		// присвоение process.env.TZ в файле теста, поэтому пояс фиксируется
		// здесь: воркеры Vitest получают его в окружении до старта.
		env: {
			TZ: "UTC",
		},
	},
});
