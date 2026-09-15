/**
 * Форматирование идентификаторов и копирование в буфер обмена.
 *
 * Сокращение UUID - `3f6d2d8e…a3b4c`: первые 8 и последние 5 шестнадцатеричных
 * цифр. Полное значение всегда доступно в `title` и на странице детали.
 *
 * Реактивное состояние «скопировано» живёт в слое состояния
 * (`src/lib/state/clipboard.svelte.ts`), потому что руны работают только в
 * файлах `.svelte.ts`. Здесь - чистые функции.
 */

export const ELLIPSIS = "\u2026";

const UUID_HEX_LENGTH = 32;
const HEAD_LENGTH = 8;
const TAIL_LENGTH = 5;

export function uuidHex(uuid: string): string {
	return uuid.replaceAll("-", "");
}

/** `3f6d2d8e…a3b4c`; строки не-UUID возвращаются как есть. */
export function shortId(uuid: string | undefined): string {
	if (!uuid) return "";

	const hex = uuidHex(uuid);
	if (hex.length !== UUID_HEX_LENGTH) return uuid;

	return `${hex.slice(0, HEAD_LENGTH)}${ELLIPSIS}${hex.slice(-TAIL_LENGTH)}`;
}

/**
 * Копирует текст в буфер обмена.
 *
 * Сначала современный `navigator.clipboard` (работает только в защищённом
 * контексте), затем запасной путь через временный `textarea` и
 * `document.execCommand("copy")` - он нужен для доступа по http с машины,
 * которая не считается защищённым контекстом. Возвращает фактический исход,
 * чтобы интерфейс не показывал «скопировано» при неудаче.
 */
export async function copyText(text: string): Promise<boolean> {
	if (text.length === 0) return false;

	try {
		if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
			await navigator.clipboard.writeText(text);
			return true;
		}
	} catch {
		// Пробуем запасной путь.
	}

	try {
		if (typeof document === "undefined") return false;

		const area = document.createElement("textarea");
		area.value = text;
		// Элемент не должен влиять на раскладку: позиционируем вне потока.
		area.setAttribute("readonly", "");
		area.style.position = "fixed";
		area.style.top = "-1000px";
		area.style.opacity = "0";
		document.body.appendChild(area);
		area.select();

		const copied = document.execCommand("copy");
		area.remove();
		return copied;
	} catch {
		return false;
	}
}

/**
 * Усечение посередине: для длинных URL и хешей начало и конец информативнее
 * хвоста. `limit` - общая длина результата вместе с многоточием.
 */
export function middleTruncate(value: string, limit = 44): string {
	if (value.length <= limit) return value;
	const head = Math.ceil(limit * 0.6);
	const tail = limit - head;
	return `${value.slice(0, head)}${ELLIPSIS}${value.slice(value.length - tail)}`;
}
