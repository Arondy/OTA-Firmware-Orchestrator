import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * Механическая проверка «жёстких запретов» из 00-CONTEXT §10.
 *
 * Эти правила легко нарушить по невнимательности и трудно заметить на ревью,
 * поэтому они закреплены тестом: проверка запускается в `bun run test` и падает
 * с указанием файла и строки. Запрещённые последовательности собираются через
 * `String.fromCharCode` и экранированные диапазоны, чтобы сам этот файл проверку
 * не проваливал.
 */

// Vitest запускается из корня пакета, поэтому путь берём от рабочей директории:
// `import.meta.url` в jsdom-окружении оказывается ненадёжным основанием для URL.
const SRC_ROOT = join(process.cwd(), "src");

const EM_DASH = String.fromCharCode(0x2014);
const EN_DASH = String.fromCharCode(0x2013);

const EMOJI_PATTERN = new RegExp(
	[
		"[\\u{1F300}-\\u{1FAFF}]",
		"[\\u{2600}-\\u{27BF}]",
		"[\\u{1F000}-\\u{1F0FF}]",
		"[\\u{FE0F}]",
		"[\\u{2190}-\\u{21FF}]",
	].join("|"),
	"u",
);

const HEX_COLOR_PATTERN = /#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})\b/;

interface SourceFile {
	path: string;
	relative: string;
	text: string;
	lines: string[];
}

function walk(directory: string): string[] {
	const result: string[] = [];

	for (const entry of readdirSync(directory)) {
		if (entry === "node_modules" || entry === ".svelte-kit" || entry === "build") continue;
		const full = join(directory, entry);
		if (statSync(full).isDirectory()) {
			result.push(...walk(full));
			continue;
		}
		result.push(full);
	}

	return result;
}

/**
 * Этот файл исключён из собственной проверки: в нём лежат литералы запрещённых
 * последовательностей, и любая проверка нашла бы их здесь же. Файл тестовый,
 * в прод-сборку не попадает.
 */
const SELF = "project-rules.test.ts";

function sources(extension: string): SourceFile[] {
	return walk(SRC_ROOT)
		.filter((path) => path.endsWith(extension) && !path.endsWith(SELF))
		.map((path) => {
			const text = readFileSync(path, "utf8");
			return { path, relative: relative(SRC_ROOT, path), text, lines: text.split("\n") };
		});
}

/** Файлы с расширениями исходников, кроме сгенерированной схемы API. */
function allSources(): SourceFile[] {
	return [".ts", ".svelte", ".css", ".html"]
		.flatMap((extension) => sources(extension))
		.filter((file) => !file.relative.endsWith("api/schema.d.ts"));
}

/**
 * Вычищает комментарии, сохраняя переносы строк, чтобы номера строк в отчёте
 * оставались настоящими. Без этого правила находили запрещённые слова в наших же
 * пояснениях к ним: комментарий «serif запрещён» не является serif-шрифтом.
 *
 * `//` не считается началом комментария, если перед ним стоит двоеточие: так
 * адрес вида `http://...` внутри строки не маскируется под комментарий.
 */
function stripComments(source: string): string {
	let out = "";
	let index = 0;
	let blockEnd: string | undefined;

	while (index < source.length) {
		if (blockEnd) {
			if (source.startsWith(blockEnd, index)) {
				out += " ".repeat(blockEnd.length);
				index += blockEnd.length;
				blockEnd = undefined;
				continue;
			}
			out += source[index] === "\n" ? "\n" : " ";
			index += 1;
			continue;
		}

		if (source.startsWith("/*", index)) {
			out += "  ";
			index += 2;
			blockEnd = "*/";
			continue;
		}
		if (source.startsWith("<!--", index)) {
			out += "    ";
			index += 4;
			blockEnd = "-->";
			continue;
		}
		if (source.startsWith("//", index) && source[index - 1] !== ":") {
			while (index < source.length && source[index] !== "\n") {
				out += " ";
				index += 1;
			}
			continue;
		}

		out += source[index];
		index += 1;
	}

	return out;
}

function findOffenders(files: SourceFile[], predicate: (line: string) => boolean): string[] {
	const offenders: string[] = [];
	for (const file of files) {
		const scanned = stripComments(file.text).split("\n");
		scanned.forEach((line, index) => {
			if (predicate(line)) offenders.push(`${file.relative}:${index + 1}: ${line.trim()}`);
		});
	}
	return offenders;
}

describe("§10.1, §10.2: цвет и шрифт", () => {
	it("в .svelte-файлах нет hex-литералов, только токены", () => {
		const offenders = findOffenders(sources(".svelte"), (line) => HEX_COLOR_PATTERN.test(line));
		expect(offenders).toEqual([]);
	});

	it("hex-значения живут только в app.css, где объявлены токены", () => {
		const offenders = findOffenders(
			sources(".ts"),
			(line) => HEX_COLOR_PATTERN.test(line) && !line.trim().startsWith("*"),
		);
		expect(offenders).toEqual([]);
	});

	it("нет фиолетовых и индиговых утилит Tailwind: акцент один", () => {
		const banned =
			/(purple|violet|indigo|fuchsia|pink|cyan|teal|emerald|sky|blue|red|green|yellow|orange|amber|slate|gray|zinc|stone|neutral)-\d{2,3}/;
		const offenders = findOffenders(allSources(), (line) => banned.test(line));
		expect(offenders).toEqual([]);
	});

	it("нет serif-шрифта", () => {
		// `sans-serif` в стеке запасных шрифтов легален, отдельный `serif` - нет.
		const offenders = findOffenders(allSources(), (line) => /(?<!-)serif\b/.test(line));
		expect(offenders).toEqual([]);
	});

	it("нет стеклянных панелей и неоновых свечений", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/backdrop-blur|backdrop-filter|box-shadow:\s*0 0 \d+px/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§10.3: типографика и декор", () => {
	it("ни одного длинного и короткого тире в исходниках", () => {
		const offenders = findOffenders(
			allSources(),
			(line) => line.includes(EM_DASH) || line.includes(EN_DASH),
		);
		expect(offenders).toEqual([]);
	});

	it("ни одного эмодзи", () => {
		const offenders = findOffenders(allSources(), (line) => EMOJI_PATTERN.test(line));
		expect(offenders).toEqual([]);
	});

	it("нет декоративных бейджей и подсказок о прокрутке", () => {
		// Проверка регистрозависимая намеренно: `live: true` в карте статусов -
		// признак пульсирующей точки, а `beta` в тестах semver - часть версии.
		const offenders = findOffenders(allSources(), (line) =>
			/\bBETA\b|INVITE-ONLY|EARLY ACCESS|\bLIVE\b|Scroll to explore/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§10.5, §10.9: адреса и стили", () => {
	it("в API-слое нет абсолютных адресов: только относительные пути", () => {
		const offenders = findOffenders(
			sources(".ts").filter((file) => file.relative.startsWith("lib/api/")),
			(line) => /localhost:\d+|127\.0\.0\.1|http:\/\/|https:\/\//.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("localhost нигде не встречается вне dev-конфига Vite", () => {
		const offenders = findOffenders(
			allSources().filter((file) => !file.relative.endsWith("vite.config.ts")),
			(line) => /localhost:\d+|127\.0\.0\.1/.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("нет h-screen: высота только через dvh", () => {
		const offenders = findOffenders(allSources(), (line) => /\bh-screen\b/.test(line));
		expect(offenders).toEqual([]);
	});

	it("нет !important", () => {
		const offenders = findOffenders(allSources(), (line) => line.includes("!important"));
		expect(offenders).toEqual([]);
	});

	it("в .svelte нет статических inline-стилей", () => {
		// style допустим только для data-driven значений (ширина по данным):
		// утилита Tailwind не может выразить рантаймовое число. Всё статическое
		// обязано жить в классах.
		const offenders = findOffenders(
			sources(".svelte"),
			(line) => /\sstyle=/.test(line) && !line.includes("{"),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§10.6: диалоги и уведомления", () => {
	it("нет нативных alert, confirm и prompt", () => {
		const offenders = findOffenders(
			allSources(),
			(line) =>
				/(?:window\.|\b)(?:alert|confirm|prompt)\s*\(/.test(line) && !/confirmText\s*\(/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§10.12: чистота кода", () => {
	it("нет any", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/:\s*any\b|<any>|as any\b|\bany\[\]/.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("нет подавления ошибок типизации", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/@ts-ignore|@ts-expect-error|@ts-nocheck|eslint-disable/.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("в непоставляемом коде нет console.log", () => {
		const offenders = findOffenders(
			allSources().filter((file) => !file.relative.endsWith(".test.ts")),
			(line) => /console\.log\s*\(/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§2: только Svelte 5 runes", () => {
	it("нет легас-синтаксиса событий и пропсов", () => {
		const offenders = findOffenders(sources(".svelte"), (line) =>
			/\son:[a-z]|^\s*export let\b/.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("нет легас-сторов на $-подписке", () => {
		const offenders = findOffenders(
			allSources(),
			(line) => /derived\(|writable\(|readable\(|\$:\s/.test(line) && !/\$derived/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§7.2: дисциплина поллинга", () => {
	it("нет setInterval без очистки вне слоя состояния", () => {
		const offenders = findOffenders(sources(".svelte"), (line) => /setInterval\s*\(/.test(line));
		expect(offenders).toEqual([]);
	});

	it("нет слушателя прокрутки на window: он запрещён outright", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/addEventListener\(\s*["']scroll["']/.test(line),
		);
		expect(offenders).toEqual([]);
	});

	it("нет GSAP и библиотек графиков", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/\bgsap\b|ScrollTrigger|chart\.js|recharts|d3-/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});

describe("§5.2: честная пагинация", () => {
	it("в исходниках нет выдуманного общего числа и «страница N из M»", () => {
		const offenders = findOffenders(allSources(), (line) =>
			/total_count|totalCount|страница \d+ из|из \d+ страниц/.test(line),
		);
		expect(offenders).toEqual([]);
	});
});
