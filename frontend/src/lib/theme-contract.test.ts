import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * Контракт предкрасочной инициализации темы.
 *
 * Тема задаётся внешним блокирующим скриптом `static/theme-init.js`, а не
 * inline-скриптом из `<ModeWatcher />`. Причины две: в SPA-сборке с `ssr = false`
 * компонент вставляет скрипт уже после загрузки бандла (вспышку не предотвращает),
 * а inline-скрипт потребовал бы sha256-хеш в CSP.
 *
 * Алгоритм повторяет `setInitialMode` из mode-watcher, поэтому здесь закреплён
 * дрейф-тест: если библиотека переименует ключ localStorage или класс, тест
 * упадёт и скрипт придётся обновить.
 */

/** Vitest стартует из корня пакета, поэтому пути отсчитываем от него. */
function read(relativePath: string): string {
	return readFileSync(join(process.cwd(), relativePath), "utf8");
}

describe("static/theme-init.js", () => {
	const source = read("static/theme-init.js");

	it("существует и непустой", () => {
		expect(source.trim().length).toBeGreaterThan(0);
	});

	it("пользуется тем же ключом localStorage, что и mode-watcher", () => {
		expect(source).toContain("mode-watcher-mode");
	});

	it("вешает тот же класс на <html>, что и mode-watcher", () => {
		expect(source).toContain('"dark"');
	});

	it("читает системную тему через prefers-color-scheme", () => {
		expect(source).toContain("prefers-color-scheme: light");
	});

	it("задаёт color-scheme, чтобы нативные элементы браузера совпадали с темой", () => {
		expect(source).toContain("colorScheme");
	});

	it("не падает, когда localStorage недоступен", () => {
		// Оба обращения к хранилищу обёрнуты в try: приватный режим не должен ломать старт.
		expect(source.match(/try \{/g)?.length ?? 0).toBeGreaterThanOrEqual(2);
	});

	it("не использует inline-обработчиков и document.write", () => {
		expect(source).not.toContain("document.write");
	});
});

describe("защита от дрейфа mode-watcher", () => {
	const library = read("node_modules/mode-watcher/dist/mode.js");
	const source = read("static/theme-init.js");

	it("библиотека по-прежнему хранит режим под ключом mode-watcher-mode", () => {
		expect(library).toContain('modeStorageKey = "mode-watcher-mode"');
	});

	it("библиотека по-прежнему использует класс dark", () => {
		expect(library).toContain('darkClassNames = ["dark"]');
	});

	it("ключ и класс совпадают с теми, что зашиты в наш скрипт", () => {
		expect(source).toContain("mode-watcher-mode");
		expect(source).toContain('"dark"');
	});

	it("алгоритм библиотеки по-прежнему опирается на prefers-color-scheme: light", () => {
		expect(library).toContain("prefers-color-scheme: light");
	});
});

describe("src/app.html", () => {
	const source = read("src/app.html");

	it("подключает theme-init.js внешним скриптом", () => {
		expect(source).toContain('<script src="/theme-init.js"></script>');
	});

	it("инициализация темы идёт до %sveltekit.head%, то есть до таблицы стилей", () => {
		expect(source.indexOf("/theme-init.js")).toBeLessThan(source.indexOf("%sveltekit.head%"));
	});

	it("нет ни одного inline-скрипта: CSP script-src 'self' обходится без хешей", () => {
		const scripts = source.match(/<script\b[^>]*>/g) ?? [];
		expect(scripts.length).toBeGreaterThan(0);
		for (const tag of scripts) {
			expect(tag).toContain("src=");
		}
	});

	it("язык документа - русский", () => {
		expect(source).toContain('lang="ru"');
	});

	it("тёмная тема объявлена темой по умолчанию", () => {
		expect(source).toContain('<html lang="ru" class="dark">');
	});

	it("нет inline-стилей: всё через классы и токены", () => {
		expect(source).not.toContain("style=");
	});
});

describe("src/app.css", () => {
	const source = read("src/app.css");

	it("задаёт токены и для светлой, и для тёмной темы", () => {
		expect(source).toContain(":root {");
		expect(source).toContain(".dark {");
	});

	it("тёмная тема объявлена после светлой, поэтому переопределяет её", () => {
		expect(source.indexOf(".dark {")).toBeGreaterThan(source.indexOf(":root {"));
	});

	it("определяет единственный акцент", () => {
		expect(source).toContain("--accent-fill");
	});

	it("вычищает палитру Tailwind по умолчанию", () => {
		expect(source).toContain("--color-*: initial");
	});
});
