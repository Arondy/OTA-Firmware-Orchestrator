/**
 * Числа по-русски: разделитель разрядов - тонкий неразрывный пробел (U+202F),
 * дробной части - запятая. `Intl` в разных сборках ICU отдаёт то U+00A0, то
 * U+202F, поэтому разделитель нормализуется.
 */
const THIN_NO_BREAK_SPACE = "\u202f";
const NBSP = "\u00a0";

/** Гасит артефакты двоичной арифметики: 0.29 * 100 === 28.999999999999996. */
const INTEGER_EPSILON_SCALE = 1e6;

function normalizeSeparators(value: string): string {
	return value.replaceAll(NBSP, THIN_NO_BREAK_SPACE);
}

const intFormatter = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 0 });
const decimalFormatters = new Map<number, Intl.NumberFormat>();

function decimalFormatter(digits: number): Intl.NumberFormat {
	let formatter = decimalFormatters.get(digits);
	if (!formatter) {
		formatter = new Intl.NumberFormat("ru-RU", {
			minimumFractionDigits: digits,
			maximumFractionDigits: digits,
		});
		decimalFormatters.set(digits, formatter);
	}
	return formatter;
}

export function int(value: number): string {
	if (!Number.isFinite(value)) return "не число";
	return normalizeSeparators(intFormatter.format(Math.trunc(value)));
}

/**
 * Доля 0..1 как процент: целым, когда значение целое, иначе с одним знаком
 * (`95%`, `98,5%`). Явное `digits` переопределяет это правило.
 */
export function percent(fraction: number, digits?: number): string {
	if (!Number.isFinite(fraction)) return "не число";

	const scaled = Math.round(fraction * 100 * INTEGER_EPSILON_SCALE) / INTEGER_EPSILON_SCALE;
	const resolvedDigits = digits ?? (Number.isInteger(scaled) ? 0 : 1);
	return `${normalizeSeparators(decimalFormatter(resolvedDigits).format(scaled))}%`;
}

/**
 * Доля от целого. При нулевом знаменателе возвращает `undefined`, а не 0:
 * показать «0%» там, где наблюдений не было, запрещено (§5.4).
 */
export function ratio(part: number, total: number): number | undefined {
	if (!Number.isFinite(part) || !Number.isFinite(total) || total === 0) return undefined;
	return part / total;
}

const pluralRules = new Intl.PluralRules("ru-RU");

/** Русская плюрализация: формы в порядке one / few / many. */
export function plural(count: number, forms: [string, string, string]): string {
	const category = pluralRules.select(Math.abs(Math.trunc(count)));
	if (category === "one") return forms[0];
	if (category === "few") return forms[1];
	return forms[2];
}
