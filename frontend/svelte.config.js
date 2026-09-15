import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/**
 * Конфигурация SvelteKit.
 *
 * adapter-static в режиме SPA: ssr выключен в src/routes/+layout.ts, поэтому весь
 * рендеринг происходит в браузере, а `fallback: "index.html"` отдаёт оболочку на любом
 * маршруте. Это то, что в контейнере раздаёт Caddy.
 *
 * `runes: true` задан явно, чтобы легас-синтаксис (`export let`, `on:click`, stores)
 * ломал сборку, а не проезжал молча: 00-CONTEXT §2 требует только runes. В
 * vite-plugin-svelte 7 эти опции живут в корне конфига, а не в разделе `vitePlugin`.
 *
 * @type {import('@sveltejs/kit').Config}
 */
const config = {
	preprocess: vitePreprocess(),
	compilerOptions: {
		runes: true,
	},
	kit: {
		// Политика безопасности собирается на этапе сборки: SvelteKit сам
		// досчитывает sha256 своего inline-бутстрапа и кладёт политику в
		// <meta http-equiv>. Наших inline-скриптов в приложении нет (тема
		// ставится внешним static/theme-init.js), поэтому хеш ровно один и он
		// пересчитывается автоматически. `frame-ancestors` в meta-теге
		// игнорируется по спецификации, его добавит Caddy заголовком.
		csp: {
			mode: "hash",
			directives: {
				"default-src": ["'self'"],
				"script-src": ["'self'"],
				"style-src": ["'self'", "'unsafe-inline'"],
				"img-src": ["'self'", "data:"],
				"font-src": ["'self'"],
				"connect-src": ["'self'"],
				"object-src": ["'none'"],
				"base-uri": ["'self'"],
				"form-action": ["'self'"],
				"frame-ancestors": ["'none'"],
			},
		},
		adapter: adapter({
			pages: "build",
			assets: "build",
			fallback: "index.html",
			precompress: false,
			strict: true,
		}),
	},
};

export default config;
