import { ApiError } from "$lib/api/errors";
import { toast } from "./toast";

/**
 * Единая точка мутаций (§7.6): флаги «выполняется», блокировка повторной
 * отправки, один тост на успех и инлайн-ошибка вместо тоста на сбой.
 */
class MutationRegistry {
	#pending = $state<Record<string, boolean>>({});

	isPending(key: string): boolean {
		return this.#pending[key] === true;
	}

	async run<T>(
		key: string,
		action: () => Promise<T>,
		options: { notify?: boolean } = {},
	): Promise<T | undefined> {
		if (this.#pending[key]) return undefined;

		this.#pending[key] = true;
		try {
			return await action();
		} catch (error) {
			// `notify: false` - вызов сам решает, как показать ошибку: например,
			// действия кампании переводят сообщение сервера своими текстами.
			if (error instanceof ApiError && options.notify !== false) toast.apiError(error, key);
			throw error;
		} finally {
			this.#pending[key] = false;
		}
	}
}

export const mutation = new MutationRegistry();
