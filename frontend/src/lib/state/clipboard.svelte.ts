/**
 * Реактивная половина копирования: чистая `copyText` живёт в `format/id.ts`,
 * здесь - общее на приложение состояние «скопировано» на 1,5 с, чтобы две
 * кнопки в разных строках таблицы не подсвечивались одновременно.
 */
import { copyText } from "$lib/format/id";

export const COPY_FEEDBACK_MS = 1500;
export const COPIED_LABEL = "скопировано";

class Clipboard {
	copiedKey = $state<string | undefined>(undefined);

	#timer: ReturnType<typeof setTimeout> | undefined;

	async copy(key: string, text: string): Promise<boolean> {
		const copied = await copyText(text);

		clearTimeout(this.#timer);
		this.#timer = undefined;

		if (!copied) {
			this.copiedKey = undefined;
			return false;
		}

		this.copiedKey = key;
		this.#timer = setTimeout(() => {
			this.copiedKey = undefined;
			this.#timer = undefined;
		}, COPY_FEEDBACK_MS);

		return true;
	}

	isCopied(key: string): boolean {
		return this.copiedKey === key;
	}
}

export const clipboard = new Clipboard();
