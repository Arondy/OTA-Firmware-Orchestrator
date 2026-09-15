/**
 * Реестр перезагрузок для глобальной кнопки «Обновить» в топбаре: экран
 * регистрирует свои ресурсы, кнопка дёргает их все и недоступна, пока идёт
 * хотя бы один запрос.
 */
class RefreshRegistry {
	#entries = new Set<() => Promise<void>>();
	#running = $state(0);

	get busy(): boolean {
		return this.#running > 0;
	}

	register(refresh: () => Promise<void>): () => void {
		this.#entries.add(refresh);
		return () => {
			this.#entries.delete(refresh);
		};
	}

	async refreshAll(): Promise<void> {
		const entries = [...this.#entries];
		if (entries.length === 0) return;

		this.#running += 1;
		try {
			await Promise.all(entries.map((refresh) => refresh()));
		} finally {
			this.#running -= 1;
		}
	}
}

export const refreshRegistry = new RefreshRegistry();
