/*
 * Предкрасочная инициализация темы.
 *
 * Зачем отдельный файл, а не `<ModeWatcher />` из mode-watcher:
 *   1. Приложение собирается adapter-static с `ssr = false`, то есть серверного
 *      рендера нет вовсе. Компонент `ModeWatcher` вставляет свой скрипт через
 *      svelte:head уже из браузера, после загрузки бандла, - предотвратить вспышку
 *      не той темы он в такой сборке физически не может.
 *   2. Это inline-скрипт. В проде приложение раздаётся за Caddy с
 *      `script-src 'self'`, и inline-скрипт потребовал бы sha256-хеш в CSP, который
 *      молча ломается при каждом обновлении зависимости.
 *
 * Поэтому здесь - маленький внешний блокирующий скрипт, который подключается в <head>
 * до таблицы стилей и до первого кадра. Ни одного inline-скрипта в приложении нет,
 * CSP остаётся строгой.
 *
 * Алгоритм повторяет `setInitialMode` из mode-watcher@1.1.0 с нашей конфигурацией
 * (defaultMode: "system", darkClassNames: ["dark"], lightClassNames: []), чтобы после
 * гидрации состояние библиотеки и уже проставленный класс не разошлись. Ключ
 * localStorage и имя класса - публичный контракт mode-watcher; тест
 * src/lib/theme-contract.test.ts падает, если библиотека их переименует.
 */
(() => {
	const MODE_KEY = "mode-watcher-mode";
	const DARK_CLASS = "dark";
	const DEFAULT_MODE = "system";

	const root = document.documentElement;

	let stored = null;
	try {
		stored = localStorage.getItem(MODE_KEY);
	} catch {
		// Приватный режим или заблокированное хранилище: работаем от системной темы.
		stored = null;
	}

	const mode = stored === null || stored === "" ? DEFAULT_MODE : stored;
	const prefersLight =
		typeof window.matchMedia === "function" &&
		window.matchMedia("(prefers-color-scheme: light)").matches;

	const light = mode === "light" || (mode === "system" && prefersLight);

	root.classList.toggle(DARK_CLASS, !light);
	root.style.colorScheme = light ? "light" : "dark";

	try {
		localStorage.setItem(MODE_KEY, mode);
	} catch {
		// Не критично: значение просто не сохранится между сессиями.
	}
})();
