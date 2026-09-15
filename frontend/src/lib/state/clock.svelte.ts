/**
 * Общие часы для относительных подписей: один таймер на приложение вместо
 * таймера на строку таблицы. Запускает и останавливает корневой макет.
 */
export const RELATIVE_TICK_MS = 30_000;

class Clock {
	now = $state(Date.now());

	#timer: ReturnType<typeof setInterval> | undefined;

	#onVisibilityChange = (): void => {
		if (document.hidden) {
			this.#clearTimer();
			return;
		}
		this.now = Date.now();
		this.start();
	};

	#clearTimer(): void {
		if (this.#timer === undefined) return;
		clearInterval(this.#timer);
		this.#timer = undefined;
	}

	start(): void {
		if (this.#timer !== undefined) return;
		this.#timer = setInterval(() => {
			this.now = Date.now();
		}, RELATIVE_TICK_MS);
		document.addEventListener("visibilitychange", this.#onVisibilityChange);
	}

	stop(): void {
		this.#clearTimer();
		document.removeEventListener("visibilitychange", this.#onVisibilityChange);
	}
}

export const clock = new Clock();
