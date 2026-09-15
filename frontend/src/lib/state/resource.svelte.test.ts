import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "$lib/api/errors";
import { apiAvailability } from "./api-availability.svelte";
import { createResource, withJitter } from "./resource.svelte";

/**
 * Ресурс тестируется поддельными таймерами: проверяются ровно те гарантии,
 * которые от него требуют экраны (00-CONTEXT §7.2).
 */

const INTERVAL = 10_000;

function setDocumentHidden(hidden: boolean): void {
	Object.defineProperty(document, "hidden", { configurable: true, get: () => hidden });
	Object.defineProperty(document, "visibilityState", {
		configurable: true,
		get: () => (hidden ? "hidden" : "visible"),
	});
	document.dispatchEvent(new Event("visibilitychange"));
}

/** Прокручивает таймеры и одновременно пропускает накопившиеся микрозадачи. */
async function advance(ms: number): Promise<void> {
	await vi.advanceTimersByTimeAsync(ms);
}

function serverError(status = 500): ApiError {
	return new ApiError({ status, method: "GET", url: "/api/v1/test" });
}

function networkError(): ApiError {
	return new ApiError({ status: 0, method: "GET", url: "/api/v1/test" });
}

beforeEach(() => {
	vi.useFakeTimers();
	setDocumentHidden(false);
	apiAvailability.reset();
});

afterEach(() => {
	vi.useRealTimers();
	vi.restoreAllMocks();
});

describe("createResource: первичная загрузка", () => {
	it("загружает данные сразу при создании", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher);

		expect(resource.pending).toBe(true);
		await advance(0);

		expect(fetcher).toHaveBeenCalledTimes(1);
		expect(resource.data).toBe("payload");
		expect(resource.pending).toBe(false);
		expect(resource.refreshing).toBe(false);
		expect(resource.error).toBeUndefined();
		resource.destroy();
	});

	it("передаёт в загрузчик AbortSignal", async () => {
		let received: AbortSignal | undefined;
		const resource = createResource("test", async (signal) => {
			received = signal;
			return 1;
		});
		await advance(0);

		expect(received).toBeInstanceOf(AbortSignal);
		resource.destroy();
	});

	it("записывает момент последнего успешного ответа", async () => {
		vi.setSystemTime(new Date("2026-09-12T12:00:00.000Z"));
		const resource = createResource("test", async () => "ok");
		await advance(0);

		expect(resource.lastUpdatedAt).toBe(Date.parse("2026-09-12T12:00:00.000Z"));
		resource.destroy();
	});

	it("без intervalMs запрашивает данные ровно один раз", async () => {
		const fetcher = vi.fn(async () => "once");
		const resource = createResource("test", fetcher);
		await advance(INTERVAL * 10);

		expect(fetcher).toHaveBeenCalledTimes(1);
		resource.destroy();
	});
});

describe("createResource: ошибки", () => {
	it("сохраняет прежние данные при сбое (stale-while-revalidate)", async () => {
		let fail = false;
		const resource = createResource<string>("test", async () => {
			if (fail) throw serverError();
			return "payload";
		});

		await advance(0);
		expect(resource.data).toBe("payload");

		fail = true;
		await resource.refresh();

		expect(resource.data).toBe("payload");
		expect(resource.error).toBeInstanceOf(ApiError);
		expect(resource.error?.status).toBe(500);
		resource.destroy();
	});

	it("стирает данные, когда keepPrevious выключен", async () => {
		let fail = false;
		const resource = createResource<string>(
			"test",
			async () => {
				if (fail) throw serverError();
				return "payload";
			},
			{ keepPrevious: false },
		);

		await advance(0);
		fail = true;
		await resource.refresh();

		expect(resource.data).toBeUndefined();
		expect(resource.error?.status).toBe(500);
		resource.destroy();
	});

	it("после сбоя pending снят, чтобы экран показал ошибку, а не вечный скелетон", async () => {
		const resource = createResource<string>("test", async () => {
			throw serverError(404);
		});
		await advance(0);

		expect(resource.pending).toBe(false);
		expect(resource.data).toBeUndefined();
		expect(resource.error?.status).toBe(404);
		resource.destroy();
	});

	it("обычное исключение превращается в ApiError, а не пробрасывается", async () => {
		const resource = createResource<string>("test", async () => {
			throw new TypeError("что-то сломалось в коде");
		});
		await advance(0);

		expect(resource.error).toBeInstanceOf(ApiError);
		expect(resource.error?.status).toBe(0);
		resource.destroy();
	});

	it("флаг доступности API взводится только ошибкой со статусом 0", async () => {
		const failing = createResource<string>("test", async () => {
			throw networkError();
		});
		await advance(0);
		expect(apiAvailability.down).toBe(true);
		failing.destroy();

		apiAvailability.reset();
		const answered = createResource<string>("test", async () => {
			throw serverError(500);
		});
		await advance(0);
		expect(apiAvailability.down).toBe(false);
		answered.destroy();
	});

	it("успешный ответ снимает флаг недоступности", async () => {
		let fail = true;
		const resource = createResource<string>("test", async () => {
			if (fail) throw networkError();
			return "ok";
		});
		await advance(0);
		expect(apiAvailability.down).toBe(true);

		fail = false;
		await resource.refresh();
		expect(apiAvailability.down).toBe(false);
		expect(apiAvailability.lastSuccessAt).toBeTypeOf("number");
		resource.destroy();
	});

	it("вызывает onSettled и на успехе, и на сбое", async () => {
		const onSettled = vi.fn();
		let fail = false;
		const resource = createResource<string>(
			"test",
			async () => {
				if (fail) throw serverError();
				return "payload";
			},
			{ onSettled },
		);

		await advance(0);
		expect(onSettled).toHaveBeenCalledWith({ ok: true, data: "payload" });

		fail = true;
		await resource.refresh();
		expect(onSettled).toHaveBeenCalledTimes(2);
		expect(onSettled.mock.calls[1][0].ok).toBe(false);
		resource.destroy();
	});
});

describe("createResource: отмена", () => {
	it("отменяет незавершённый запрос при destroy", async () => {
		const aborts: AbortSignal[] = [];
		const resource = createResource<string>("test", (signal) => {
			aborts.push(signal);
			return new Promise<string>((resolve, reject) => {
				signal.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
				setTimeout(() => resolve("payload"), 1000);
			});
		});

		await advance(0);
		expect(aborts).toHaveLength(1);
		expect(aborts[0].aborted).toBe(false);

		resource.destroy();
		expect(aborts[0].aborted).toBe(true);

		await advance(5000);
		expect(resource.data).toBeUndefined();
		expect(resource.error).toBeUndefined();
	});

	it("отменяет предыдущий запрос при новом", async () => {
		const aborts: AbortSignal[] = [];
		const resource = createResource<string>("test", (signal) => {
			aborts.push(signal);
			return new Promise<string>((resolve, reject) => {
				signal.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
				setTimeout(() => resolve(`payload-${aborts.length}`), 500);
			});
		});

		await advance(0);
		const first = aborts[0];

		void resource.refresh();
		await advance(0);

		expect(first.aborted).toBe(true);
		expect(aborts).toHaveLength(2);

		await advance(600);
		expect(resource.data).toBe("payload-2");
		expect(resource.error).toBeUndefined();
		resource.destroy();
	});

	it("после destroy запросы больше не уходят", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, { intervalMs: INTERVAL });
		await advance(0);

		resource.destroy();
		await advance(INTERVAL * 5);

		expect(fetcher).toHaveBeenCalledTimes(1);
	});

	it("повторный destroy безопасен", async () => {
		const resource = createResource("test", async () => "payload");
		await advance(0);

		expect(() => {
			resource.destroy();
			resource.destroy();
		}).not.toThrow();
	});
});

describe("createResource: поллинг и видимость вкладки", () => {
	it("повторяет запрос по интервалу", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, { intervalMs: INTERVAL });
		await advance(0);
		expect(fetcher).toHaveBeenCalledTimes(1);

		// Дрожание ±15%: за 0.8 интервала тика ещё не было.
		await advance(INTERVAL * 0.8);
		expect(fetcher).toHaveBeenCalledTimes(1);

		await advance(INTERVAL * 0.4);
		expect(fetcher.mock.calls.length).toBeGreaterThanOrEqual(2);
		resource.destroy();
	});

	it("в скрытой вкладке запросов нет вовсе", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, { intervalMs: INTERVAL });
		await advance(0);
		expect(fetcher).toHaveBeenCalledTimes(1);

		setDocumentHidden(true);
		await advance(INTERVAL * 6);
		expect(fetcher).toHaveBeenCalledTimes(1);
		resource.destroy();
	});

	it("при возврате вкладки догружает данные сразу, не дожидаясь тика", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, { intervalMs: INTERVAL });
		await advance(0);

		setDocumentHidden(true);
		await advance(INTERVAL * 3);
		expect(fetcher).toHaveBeenCalledTimes(1);

		setDocumentHidden(false);
		await advance(0);
		expect(fetcher).toHaveBeenCalledTimes(2);
		resource.destroy();
	});

	it("одноразовый ресурс не перезапрашивается при возврате вкладки", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher);
		await advance(0);
		expect(fetcher).toHaveBeenCalledTimes(1);

		setDocumentHidden(true);
		await advance(5000);
		setDocumentHidden(false);
		await advance(5000);

		// Списки не опрашиваются (§7.2): фокус вкладки не должен плодить запросы.
		expect(fetcher).toHaveBeenCalledTimes(1);
		resource.destroy();
	});

	it("фоновая перезагрузка помечается refreshing, а не pending", async () => {
		let resolveSecond: ((value: string) => void) | undefined;
		let calls = 0;
		const resource = createResource<string>(
			"test",
			() => {
				calls += 1;
				if (calls === 1) return Promise.resolve("first");
				return new Promise<string>((resolve) => {
					resolveSecond = resolve;
				});
			},
			{ intervalMs: INTERVAL },
		);

		await advance(0);
		expect(resource.data).toBe("first");

		void resource.refresh();
		await advance(0);
		expect(resource.refreshing).toBe(true);
		expect(resource.pending).toBe(false);

		resolveSecond?.("second");
		await advance(0);
		expect(resource.refreshing).toBe(false);
		expect(resource.data).toBe("second");
		resource.destroy();
	});
});

describe("createResource: гейт enabled()", () => {
	it("первичную загрузку не блокирует, иначе предикат не на чем считать", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, {
			intervalMs: INTERVAL,
			enabled: () => false,
		});
		await advance(0);

		expect(fetcher).toHaveBeenCalledTimes(1);
		resource.destroy();
	});

	it("останавливает поллинг, пока предикат ложен, и держит таймер живым", async () => {
		let enabled = false;
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, {
			intervalMs: INTERVAL,
			enabled: () => enabled,
		});

		await advance(0);
		await advance(INTERVAL * 3);
		expect(fetcher).toHaveBeenCalledTimes(1);

		enabled = true;
		// Тики во время гейта не исчезают, а переносятся, поэтому за окно в 1,2
		// интервала их может пройти больше одного. Существенно то, что запросы
		// возобновились, а не их точное число.
		await advance(INTERVAL * 1.2);
		expect(fetcher.mock.calls.length).toBeGreaterThanOrEqual(2);
		resource.destroy();
	});

	it("refresh() обходит предикат: это явное действие оператора", async () => {
		const fetcher = vi.fn(async () => "payload");
		const resource = createResource("test", fetcher, {
			intervalMs: INTERVAL,
			enabled: () => false,
		});
		await advance(0);

		await resource.refresh();
		expect(fetcher).toHaveBeenCalledTimes(2);
		resource.destroy();
	});
});

describe("withJitter", () => {
	it("остаётся в пределах ±15% от интервала", () => {
		for (let index = 0; index < 500; index += 1) {
			const value = withJitter(INTERVAL);
			expect(value).toBeGreaterThanOrEqual(INTERVAL * 0.85);
			expect(value).toBeLessThanOrEqual(INTERVAL * 1.15);
		}
	});

	it("при нулевом коэффициенте дрожания нет", () => {
		expect(withJitter(INTERVAL, 0)).toBe(INTERVAL);
	});

	it("реально меняет задержку, иначе панели стреляли бы синхронно", () => {
		const values = new Set<number>();
		for (let index = 0; index < 50; index += 1) values.add(withJitter(INTERVAL));
		expect(values.size).toBeGreaterThan(1);
	});
});

describe("createResource: notifyAvailability", () => {
	it("сбой молчащего ресурса не взводит флаг недоступности", async () => {
		const quiet = createResource<string>(
			"test",
			async () => {
				throw networkError();
			},
			{ notifyAvailability: false },
		);
		await advance(0);

		expect(quiet.error).toBeInstanceOf(ApiError);
		expect(apiAvailability.down).toBe(false);
		quiet.destroy();
	});

	it("успех молчащего ресурса не снимает уже поднятый флаг", async () => {
		apiAvailability.noteFailure(networkError(), Date.now());
		expect(apiAvailability.down).toBe(true);

		const quiet = createResource<string>("test", async () => "ok", {
			notifyAvailability: false,
		});
		await advance(0);

		expect(quiet.data).toBe("ok");
		expect(apiAvailability.down).toBe(true);
		quiet.destroy();
	});
});
