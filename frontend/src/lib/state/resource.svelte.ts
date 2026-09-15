/**
 * Единственная примитивная обёртка над загрузкой данных (00-CONTEXT §7.2).
 * Все экраны пользуются только ею: вариантов паттерна в проекте нет.
 *
 * Что гарантировано:
 *   - на каждый запрос свой `AbortController`, предыдущий отменяется;
 *   - поллинг останавливается, когда вкладка скрыта, и сразу догружает данные,
 *     когда она снова видна;
 *   - интервалы дрожат на ±15%, чтобы панели не стреляли синхронно;
 *   - ошибка не стирает уже показанные данные (stale-while-revalidate);
 *   - таймеры и слушатели всегда убираются в `destroy()`.
 *
 * `load`-функции SvelteKit для данных не используются намеренно: SSR выключен,
 * они только размазали бы логику по двум местам.
 */
import { onDestroy } from "svelte";
import { ApiError } from "$lib/api/errors";
import { apiAvailability } from "./api-availability.svelte";

export interface ResourceSettled<T> {
	ok: boolean;
	data?: T;
	error?: ApiError;
}

export interface ResourceOptions<T> {
	/**
	 * Период поллинга в миллисекундах. `0` или отсутствие - загрузить один раз.
	 * Фактическая задержка каждый тик пересчитывается с дрожанием ±15%.
	 */
	intervalMs?: number;
	/**
	 * Гейт поллинга: пока функция возвращает `false`, новые запросы не уходят,
	 * но таймер остаётся живым и проверка повторяется. Первичную загрузку не
	 * блокирует - иначе данные, от которых зависит сам предикат, не получить.
	 */
	enabled?: () => boolean;
	/** Сохранять прежние данные при ошибке. По умолчанию `true`. */
	keepPrevious?: boolean;
	/** Вызывается после каждого завершившегося запроса (успех или ошибка). */
	onSettled?: (settled: ResourceSettled<T>) => void;
	/**
	 * Отмечать ли исход запроса в глобальном флаге доступности API
	 * (`apiAvailability`). По умолчанию `true`. Выключает обёртка, чей загрузчик
	 * сам управляет флагом: например, агрегатор кампаний отмечает каждый
	 * подзапрос отдельно, а его собственный успешный «пустой» результат не
	 * должен объявлять API доступным (иначе баннер «API недоступен» мигает,
	 * когда агрегатор проглотил все ошибки и вернул пустой массив).
	 */
	notifyAvailability?: boolean;
}

export interface Resource<T> {
	/** Ключ ресурса: для отладки и для склейки ключей уведомлений. */
	readonly key: string;
	/** Последние успешные данные. Ошибкой не стираются, если `keepPrevious`. */
	readonly data: T | undefined;
	readonly error: ApiError | undefined;
	/** Идёт первая загрузка, данных на экране ещё нет. */
	readonly pending: boolean;
	/** Идёт фоновая перезагрузка, данные уже показаны. */
	readonly refreshing: boolean;
	/** Момент последнего успешного ответа, для подписи «обновлено N с назад». */
	readonly lastUpdatedAt: number | undefined;
	/** Принудительно перезагрузить, игнорируя предикат `enabled()`. */
	refresh(): Promise<void>;
	/**
	 * Подставить данные извне - например, тело ответа мутации, - не делая
	 * запроса. Ошибку сбрасывает: данные свежие и достоверные.
	 */
	write(data: T): void;
	/** Отменить запрос, снять таймеры и слушатели. */
	destroy(): void;
}

const ABORT_REASON = "запрос отменён: ресурс перезапущен или уничтожен";

function isAbort(error: unknown): boolean {
	return (
		typeof error === "object" &&
		error !== null &&
		"name" in error &&
		(error as { name?: unknown }).name === "AbortError"
	);
}

/** Дрожание интервала: ±ratio от базового значения. */
export function withJitter(intervalMs: number, ratio = 0.15): number {
	const spread = intervalMs * ratio;
	return Math.round(intervalMs - spread + Math.random() * spread * 2);
}

function hasDocument(): boolean {
	return typeof document !== "undefined";
}

class ResourceImpl<T> implements Resource<T> {
	readonly key: string;

	data = $state<T | undefined>(undefined);
	error = $state<ApiError | undefined>(undefined);
	pending = $state(true);
	refreshing = $state(false);
	lastUpdatedAt = $state<number | undefined>(undefined);

	#fetcher: (signal: AbortSignal) => Promise<T>;
	#intervalMs: number;
	#enabled: (() => boolean) | undefined;
	#keepPrevious: boolean;
	#onSettled: ((settled: ResourceSettled<T>) => void) | undefined;
	#notifyAvailability: boolean;

	#controller: AbortController | undefined;
	#timer: ReturnType<typeof setTimeout> | undefined;
	#destroyed = false;
	#settledOnce = false;

	constructor(
		key: string,
		fetcher: (signal: AbortSignal) => Promise<T>,
		options: ResourceOptions<T> = {},
	) {
		this.key = key;
		this.#fetcher = fetcher;
		this.#intervalMs = options.intervalMs ?? 0;
		this.#enabled = options.enabled;
		this.#keepPrevious = options.keepPrevious ?? true;
		this.#onSettled = options.onSettled;
		this.#notifyAvailability = options.notifyAvailability ?? true;

		if (hasDocument()) {
			document.addEventListener("visibilitychange", this.#onVisibilityChange);
		}

		// Вне компонента (юнит-тесты, сервисный код) `onDestroy` бросает исключение;
		// тогда ресурс закрывается явным вызовом `destroy()`.
		try {
			onDestroy(() => this.destroy());
		} catch {
			// Контекста компонента нет - автоочистку не регистрируем.
		}

		void this.#run(true);
	}

	#onVisibilityChange = (): void => {
		if (this.#destroyed) return;

		if (document.hidden) {
			// Таймер снимаем целиком: пока вкладка скрыта, запросов не будет вообще.
			clearTimeout(this.#timer);
			this.#timer = undefined;
			return;
		}

		// Вкладка снова видна: догружаем сразу, не дожидаясь следующего тика.
		// Одноразовые ресурсы (intervalMs = 0) не трогаем: списки по §7.2 не
		// опрашиваются, их обновляет явный refresh() после мутаций.
		if (this.#intervalMs > 0) void this.#run(false);
	};

	#scheduleNext(): void {
		if (this.#destroyed || this.#intervalMs <= 0) return;

		clearTimeout(this.#timer);
		this.#timer = undefined;

		// В скрытой вкладке таймер не заводим: его запуск обеспечит visibilitychange.
		if (hasDocument() && document.hidden) return;

		this.#timer = setTimeout(() => {
			this.#timer = undefined;
			void this.#run(false);
		}, withJitter(this.#intervalMs));
	}

	async #run(isInitial: boolean): Promise<void> {
		if (this.#destroyed) return;

		const gate = this.#enabled;
		if (!isInitial && gate && !gate()) {
			// Поллинг выключен предикатом, но таймер жив: состояние может измениться.
			this.#scheduleNext();
			return;
		}

		const controller = new AbortController();
		const previous = this.#controller;
		this.#controller = controller;
		previous?.abort(new DOMException(ABORT_REASON, "AbortError"));

		if (this.#settledOnce && this.data !== undefined) {
			this.refreshing = true;
		} else {
			this.pending = true;
			this.refreshing = false;
		}

		let settled: ResourceSettled<T> | undefined;
		try {
			const data = await this.#fetcher(controller.signal);

			if (this.#destroyed || this.#controller !== controller) return;

			const now = Date.now();
			this.data = data;
			this.error = undefined;
			this.lastUpdatedAt = now;
			if (this.#notifyAvailability) apiAvailability.noteSuccess(now);
			settled = { ok: true, data };
		} catch (error) {
			if (this.#destroyed || this.#controller !== controller) return;

			// Отмена инициатором - штатная ситуация, а не сбой.
			if (isAbort(error)) return;

			const apiError =
				error instanceof ApiError
					? error
					: new ApiError({
							status: 0,
							method: "GET",
							url: this.key,
							serverMessage: undefined,
							cause: error,
						});

			this.error = apiError;
			if (!this.#keepPrevious) this.data = undefined;
			if (this.#notifyAvailability) apiAvailability.noteFailure(apiError, Date.now());
			settled = { ok: false, error: apiError };
		} finally {
			if (!this.#destroyed && this.#controller === controller) {
				this.#settledOnce = true;
				this.pending = false;
				this.refreshing = false;
				this.#scheduleNext();
			}
		}

		if (settled) this.#onSettled?.(settled);
	}

	write(data: T): void {
		this.data = data;
		this.error = undefined;
		this.lastUpdatedAt = Date.now();
	}

	async refresh(): Promise<void> {
		if (this.#destroyed) return;
		clearTimeout(this.#timer);
		this.#timer = undefined;
		await this.#run(true);
	}

	destroy(): void {
		if (this.#destroyed) return;
		this.#destroyed = true;

		clearTimeout(this.#timer);
		this.#timer = undefined;
		this.#controller?.abort(new DOMException(ABORT_REASON, "AbortError"));
		this.#controller = undefined;

		if (hasDocument()) {
			document.removeEventListener("visibilitychange", this.#onVisibilityChange);
		}
	}
}

/**
 * Создаёт ресурс. Внутри компонента ресурс уничтожается сам через `onDestroy`;
 * вне компонента нужно вызвать `destroy()` вручную.
 */
export function createResource<T>(
	key: string,
	fetcher: (signal: AbortSignal) => Promise<T>,
	options: ResourceOptions<T> = {},
): Resource<T> {
	return new ResourceImpl<T>(key, fetcher, options);
}
