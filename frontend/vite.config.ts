import tailwindcss from "@tailwindcss/vite";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";
import { readFileSync } from "node:fs";

/**
 * Адрес бэкенда для dev-прокси. Единственное место в проекте, где встречается
 * localhost:8080: код приложения ходит только на относительные пути (00-CONTEXT §5),
 * а в проде их проксирует Caddy.
 */
const DEV_UPSTREAM = process.env.VITE_DEV_UPSTREAM ?? "http://localhost:8080";

/* Адрес rollout-контроллера для dev-прокси: его health-эндпоинт доступен
 * браузеру через префикс /controller (в контейнере его проксирует Caddy). */
const CONTROLLER_DEV_UPSTREAM = process.env.VITE_DEV_CONTROLLER_UPSTREAM ?? "http://localhost:8090";

/* Версия и момент сборки приходят в рантайм из сборки, а не хардкодом:
 * футер сайдбара показывает настоящие значения. */
const pkg = JSON.parse(readFileSync(new URL("./package.json", import.meta.url), "utf8"));

export default defineConfig({
	define: {
		__APP_VERSION__: JSON.stringify(pkg.version),
		__BUILD_TIME__: JSON.stringify(new Date().toISOString()),
	},
	plugins: [sveltekit(), tailwindcss()],
	server: {
		proxy: {
			"/api": DEV_UPSTREAM,
			"/healthz": DEV_UPSTREAM,
			"/controller": {
				target: CONTROLLER_DEV_UPSTREAM,
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/controller/, ""),
			},
		},
	},
	// То же самое для `bun run preview`: проверка прод-сборки идёт через
	// относительные пути, как и в контейнере, где их проксирует Caddy.
	preview: {
		proxy: {
			"/api": DEV_UPSTREAM,
			"/healthz": DEV_UPSTREAM,
			"/controller": {
				target: CONTROLLER_DEV_UPSTREAM,
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/controller/, ""),
			},
		},
	},
	build: {
		// Отчёт о размере бандла в кибибайтах.
		reportCompressedSize: true,
		chunkSizeWarningLimit: 250,
	},
});
