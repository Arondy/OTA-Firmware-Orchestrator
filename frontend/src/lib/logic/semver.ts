/**
 * Semver без внешних зависимостей.
 *
 * Регулярное выражение взято один в один из `openapi.yaml` (поля
 * `current_version`, `fw_version`), поэтому клиент считает валидным ровно то,
 * что примет сервер. Сравнение следует SemVer 2.0.0:
 *   - метаданные сборки (`+build`) в precedence не участвуют;
 *   - версия без pre-release старше версии с pre-release;
 *   - числовые идентификаторы pre-release сравниваются как числа и считаются
 *     младше буквенно-цифровых.
 *
 * Сервер для решения о выдаче обновления использует
 * `semver1.GreaterThanEqual(semver2)` (Masterminds/semver), то есть обновление
 * не выдаётся при `current_version >= target`. Это `isAtLeast` ниже.
 */

const SEMVER_PATTERN =
	/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/;

const NUMERIC_IDENTIFIER = /^(0|[1-9]\d*)$/;

export interface SemVer {
	major: number;
	minor: number;
	patch: number;
	/** Идентификаторы pre-release по точкам; пустой массив, если их нет. */
	prerelease: string[];
	/** Идентификаторы метаданных сборки; на сравнение не влияют. */
	build: string[];
	/** Исходная строка. */
	raw: string;
}

/** Разбирает строку версии. `undefined`, если строка не проходит валидацию API. */
export function parseSemver(input: string): SemVer | undefined {
	const match = SEMVER_PATTERN.exec(input);
	if (!match) return undefined;

	const [, major, minor, patch, prerelease, build] = match;

	return {
		major: Number.parseInt(major, 10),
		minor: Number.parseInt(minor, 10),
		patch: Number.parseInt(patch, 10),
		prerelease: prerelease ? prerelease.split(".") : [],
		build: build ? build.split(".") : [],
		raw: input,
	};
}

/** Проходит ли строка валидацию версии из контракта API. */
export function isSemver(input: string): boolean {
	return SEMVER_PATTERN.test(input);
}

/**
 * Сравнение числовых идентификаторов без `Number`: регулярное выражение
 * запрещает ведущие нули, поэтому сначала сравнивается длина, затем лексика.
 * Точность не теряется на сколь угодно больших номерах.
 */
function compareNumericIdentifiers(a: string, b: string): number {
	if (a.length !== b.length) return a.length < b.length ? -1 : 1;
	if (a === b) return 0;
	return a < b ? -1 : 1;
}

function comparePrereleaseIdentifiers(a: string, b: string): number {
	const aNumeric = NUMERIC_IDENTIFIER.test(a);
	const bNumeric = NUMERIC_IDENTIFIER.test(b);

	if (aNumeric && bNumeric) return compareNumericIdentifiers(a, b);
	// Числовой идентификатор всегда младше буквенно-цифрового (SemVer 2.0.0, п. 11.4.3).
	if (aNumeric) return -1;
	if (bNumeric) return 1;
	if (a === b) return 0;
	return a < b ? -1 : 1;
}

/** Сравнивает разобранные версии: -1, 0 или 1. */
export function compareParsed(a: SemVer, b: SemVer): number {
	const numeric =
		Math.sign(a.major - b.major) || Math.sign(a.minor - b.minor) || Math.sign(a.patch - b.patch);
	if (numeric !== 0) return numeric;

	if (a.prerelease.length === 0 && b.prerelease.length === 0) return 0;
	if (a.prerelease.length === 0) return 1;
	if (b.prerelease.length === 0) return -1;

	const shared = Math.min(a.prerelease.length, b.prerelease.length);
	for (let index = 0; index < shared; index += 1) {
		const result = comparePrereleaseIdentifiers(a.prerelease[index], b.prerelease[index]);
		if (result !== 0) return result;
	}

	return Math.sign(a.prerelease.length - b.prerelease.length);
}

/** Сравнивает строки версий. Бросает `RangeError`, если строка не semver. */
export function compareSemver(a: string, b: string): number {
	const left = parseSemver(a);
	const right = parseSemver(b);

	if (!left) throw new RangeError(`не удалось разобрать версию прошивки: ${a}`);
	if (!right) throw new RangeError(`не удалось разобрать версию прошивки: ${b}`);

	return compareParsed(left, right);
}

/** То же, но без исключения: `undefined`, если хотя бы одна строка не semver. */
export function tryCompareSemver(a: string, b: string): number | undefined {
	const left = parseSemver(a);
	const right = parseSemver(b);
	if (!left || !right) return undefined;
	return compareParsed(left, right);
}

/**
 * `current >= target` - ровно то условие, по которому сервер отказывает
 * в обновлении на чек-ине. `undefined`, если версия не разобрана: интерфейс в этом
 * случае обязан сказать «неизвестно», а не угадать.
 */
export function isAtLeast(current: string, target: string): boolean | undefined {
	const result = tryCompareSemver(current, target);
	return result === undefined ? undefined : result >= 0;
}

/** Сортировка по возрастанию; неразобранные строки остаются в конце. */
export function sortSemver(values: string[]): string[] {
	const parsed: { raw: string; version: SemVer }[] = [];
	const rest: string[] = [];

	for (const raw of values) {
		const version = parseSemver(raw);
		if (version) parsed.push({ raw, version });
		else rest.push(raw);
	}

	parsed.sort((a, b) => compareParsed(a.version, b.version));
	return [...parsed.map((item) => item.raw), ...rest];
}

/**
 * Ключ сортировки версий строкой: танцуем вокруг танстек-сортировки, которая
 * сравнивает значения аксессора лексически. Числовые части дополняются до
 * шести знаков, поэтому 1.10.0 старше 1.9.0; префикс pre-release уходит перед
 * релизом («.» младше «~»), неразобранные строки получают пустой ключ и
 * опускаются в начало списка по возрастанию - интерфейс всё равно подписывает
 * их «версия не разобрана».
 */
export function semverSortKey(value: string): string {
	const parsed = parseSemver(value);
	if (!parsed) return "";

	const pad = (part: number): string => String(part).padStart(6, "0");
	const core = `${pad(parsed.major)}${pad(parsed.minor)}${pad(parsed.patch)}`;
	if (parsed.prerelease.length === 0) return `${core}~`;

	const pre = parsed.prerelease
		.map((ident) => (NUMERIC_IDENTIFIER.test(ident) ? pad(Number.parseInt(ident, 10)) : ident))
		.join(".");
	return `${core}.${pre}`;
}
