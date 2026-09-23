/**
 * Форматирование дат и времени (00-CONTEXT §9). Функции чистые и принимают
 * `now`, поэтому тестируются без подделки системного времени; общий реактивный
 * тик раз в 30 с живёт в `state/clock.svelte.ts`, потому что руны доступны
 * только в `.svelte.ts`.
 */

const MONTHS_SHORT = [
	"янв",
	"фев",
	"мар",
	"апр",
	"май",
	"июн",
	"июл",
	"авг",
	"сен",
	"окт",
	"ноя",
	"дек",
] as const;

const SECOND_MS = 1000;
const MINUTE_MS = 60 * SECOND_MS;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

/** Подпись, когда сервер отдал дату, которую браузер не смог разобрать. */
export const UNPARSEABLE_DATE_LABEL = "дата не разобрана";

/** Порог, до которого относительное время не показывается вовсе. */
const JUST_NOW_MS = 10 * SECOND_MS;

/** Допустимый перекос часов: небольшие будущие отметки считаем «только что». */
const CLOCK_SKEW_MS = 60 * SECOND_MS;

const timeFormatter = new Intl.DateTimeFormat("ru-RU", {
	hour: "2-digit",
	minute: "2-digit",
	hour12: false,
});

function parseDate(iso: string | undefined): Date | undefined {
	if (!iso) return undefined;
	const time = Date.parse(iso);
	if (Number.isNaN(time)) return undefined;
	return new Date(time);
}

/** `12 сен 2026, 14:32` - формат из словаря §9, без точки после месяца и без «г.». */
export function absoluteDateTime(iso: string | undefined): string {
	const date = parseDate(iso);
	if (!date) return UNPARSEABLE_DATE_LABEL;

	const day = date.getDate();
	const month = MONTHS_SHORT[date.getMonth()];
	const year = date.getFullYear();
	return `${day} ${month} ${year}, ${timeFormatter.format(date)}`;
}

/**
 * Полное значение для атрибута `title`: локальная дата и время плюс исходный
 * ISO-момент, чтобы оператор мог сверить его с логами сервера.
 */
export function absoluteDateTimeTitle(iso: string | undefined): string {
	const date = parseDate(iso);
	if (!date) return UNPARSEABLE_DATE_LABEL;
	return `${absoluteDateTime(iso)} (UTC ${date.toISOString()})`;
}

function isPreviousCalendarDay(past: Date, reference: Date): boolean {
	const startOf = (value: Date): number =>
		new Date(value.getFullYear(), value.getMonth(), value.getDate()).getTime();
	return startOf(reference) - startOf(past) === DAY_MS;
}

/** `только что`, `12 с назад`, `5 мин назад`, `2 ч назад`, `вчера`, дальше - абсолютная дата. */
export function relativeTime(iso: string | undefined, now: number = Date.now()): string {
	const date = parseDate(iso);
	if (!date) return UNPARSEABLE_DATE_LABEL;

	const timestamp = date.getTime();
	const age = now - timestamp;

	// Отметка из будущего: небольшой перекос часов терпим, большой - показываем как есть.
	if (age < 0) {
		return age >= -CLOCK_SKEW_MS ? "только что" : absoluteDateTime(iso);
	}

	if (age < JUST_NOW_MS) return "только что";
	if (age < MINUTE_MS) return `${Math.floor(age / SECOND_MS)} с назад`;
	if (age < HOUR_MS) return `${Math.floor(age / MINUTE_MS)} мин назад`;
	if (age < DAY_MS && !isPreviousCalendarDay(date, new Date(now))) {
		return `${Math.floor(age / HOUR_MS)} ч назад`;
	}
	if (isPreviousCalendarDay(date, new Date(now))) return "вчера";

	return absoluteDateTime(iso);
}

/** Подпись «обновлено N с назад» для момента в миллисекундах эпохи. */
export function updatedAgoLabel(timestamp: number | undefined, now: number): string | undefined {
	if (timestamp === undefined) return undefined;
	return relativeTime(new Date(timestamp).toISOString(), now);
}
