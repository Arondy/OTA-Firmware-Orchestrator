/**
 * Машина батч-прогона песочницы.
 *
 * Прогон ведёт настоящий трафик через API оркестратора: регистрация устройств,
 * чек-ины и отчёты. Ни одной выдуманной строки журнала: каждая запись - это
 * факт запроса с его фактическим статусом (§10.4). Остановка рвёт все висящие
 * запросы через AbortController и честно пишет, на каком устройстве встали.
 *
 * Пределы безопасности: не больше 200 устройств и 8 параллельных запросов на
 * прогон; автоповторов на 5xx нет вовсе.
 */
import { checkinDevice, createDevice, reportUpdateAttempt } from "$lib/api/endpoints";
import { ApiError } from "$lib/api/errors";
import type { ReportInput } from "$lib/api/types";
import { mulberry32, pickBatchResult } from "$lib/domain/batch-prng";
import { shortId } from "$lib/format/id";
import { apiAvailability } from "./api-availability.svelte";

export const BATCH_MAX_DEVICES = 200;
export const BATCH_MAX_CONCURRENCY = 8;
export const LOG_LINE_CAP = 500;

export interface BatchConfig {
	count: number;
	model: string;
	startVersion: string;
	successSharePercent: number;
	delayMs: number;
	concurrency: number;
	campaignId: string;
	stageId: string;
	seed: number;
}

export type LogTone = "success" | "danger" | "neutral";

export interface LogLine {
	at: number;
	device: string;
	kind: "register" | "checkin" | "report" | "run";
	status: number | undefined;
	note: string;
	tone: LogTone;
}

export interface BatchCounters {
	registered: number;
	updated: number;
	outOfSample: number;
	success: number;
	failure: number;
	timeout: number;
	requestError: number;
}

export type BatchPhase = "idle" | "running" | "stopped" | "done";

const EMPTY_COUNTERS: BatchCounters = {
	registered: 0,
	updated: 0,
	outOfSample: 0,
	success: 0,
	failure: 0,
	timeout: 0,
	requestError: 0,
};

function stamp(at: number): string {
	const date = new Date(at);
	const pad = (value: number, length = 2) => String(value).padStart(length, "0");
	return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}`;
}

class BatchRunner {
	phase = $state<BatchPhase>("idle");
	counters = $state<BatchCounters>({ ...EMPTY_COUNTERS });
	processed = $state(0);
	total = $state(0);
	seed = $state<number | undefined>(undefined);
	log = $state<LogLine[]>([]);
	summary = $state<string | undefined>(undefined);

	#controller: AbortController | undefined;

	get running(): boolean {
		return this.phase === "running";
	}

	/** Строки журнала для отрисовки: метка времени готовится один раз. */
	get lines(): (LogLine & { time: string })[] {
		return this.log.map((line) => ({ ...line, time: stamp(line.at) }));
	}

	#push(line: LogLine): void {
		this.log.push(line);
		if (this.log.length > LOG_LINE_CAP) {
			this.log.splice(0, this.log.length - LOG_LINE_CAP);
		}
	}

	async start(config: BatchConfig): Promise<void> {
		if (this.phase === "running" || apiAvailability.down) return;

		this.#controller = new AbortController();
		const signal = this.#controller.signal;
		const rng = mulberry32(config.seed);

		this.phase = "running";
		this.counters = { ...EMPTY_COUNTERS };
		this.processed = 0;
		this.total = config.count;
		this.seed = config.seed;
		this.log = [];
		this.summary = undefined;
		this.#push({
			at: Date.now(),
			device: "-",
			kind: "run",
			status: undefined,
			note: `прогон начат: устройств ${config.count}, seed ${config.seed}, параллельность ${config.concurrency}`,
			tone: "neutral",
		});

		const queue = Array.from({ length: config.count }, (_, index) => index);
		const workers = Array.from(
			{ length: Math.min(config.concurrency, BATCH_MAX_CONCURRENCY) },
			() => this.#worker(queue, config, rng, signal),
		);
		await Promise.all(workers);

		const stopped = signal.aborted;
		this.phase = stopped ? "stopped" : "done";
		this.summary = stopped
			? `остановлено пользователем на ${this.processed} / ${this.total}`
			: `прогон завершён: ${this.processed} / ${this.total}`;
		this.#push({
			at: Date.now(),
			device: "-",
			kind: "run",
			status: undefined,
			note: this.summary,
			tone: stopped ? "danger" : "success",
		});
		this.#controller = undefined;
	}

	stop(): void {
		this.#controller?.abort();
	}

	/**
	 * Запись в журнал для одиночных запросов песочницы (регистрация, чек-ин,
	 * отчёт кнопками «Шагов»): журнал общий для батча и ручных действий,
	 * каждая строка - факт настоящего запроса с его фактическим исходом.
	 */
	noteManual(
		kind: LogLine["kind"],
		deviceId: string | undefined,
		status: number | undefined,
		note: string,
		tone: LogTone,
	): void {
		this.#push({
			at: Date.now(),
			device: deviceId ? shortId(deviceId) : "-",
			kind,
			status,
			note,
			tone,
		});
	}

	async #worker(
		queue: number[],
		config: BatchConfig,
		rng: () => number,
		signal: AbortSignal,
	): Promise<void> {
		for (;;) {
			if (signal.aborted) return;
			const index = queue.shift();
			if (index === undefined) return;

			await this.#driveOne(config, rng, signal);
			this.processed += 1;
			if (config.delayMs > 0) {
				await new Promise((resolve) => setTimeout(resolve, config.delayMs));
			}
		}
	}

	async #driveOne(config: BatchConfig, rng: () => number, signal: AbortSignal): Promise<void> {
		// 1. Регистрация: реальное устройство в Postgres.
		let deviceId: string;
		try {
			const device = await createDevice(
				{ device_model: config.model, current_version: config.startVersion },
				{ signal },
			);
			deviceId = device.id;
			this.counters.registered += 1;
			this.#push({
				at: Date.now(),
				device: shortId(device.id),
				kind: "register",
				status: 201,
				note: `модель ${config.model}, версия ${config.startVersion}`,
				tone: "neutral",
			});
		} catch (error) {
			this.#noteRequestError("register", undefined, error);
			return;
		}

		// 2. Чек-ин: сервер решает выдачу обновления.
		let updateAvailable: boolean;
		// Отчёт уходит на стадию из ответа чек-ина, а не на снятую при запуске
		// батча: длинный прогон может пережить продвижение стадии, и устройство
		// обязано сообщить ту стадию, обновление которой ему выдали.
		let stageId = config.stageId;
		try {
			const checkin = await checkinDevice(
				deviceId,
				{ current_version: config.startVersion },
				{ signal },
			);
			updateAvailable = checkin.update_available;
			stageId = checkin.stage_id ?? config.stageId;
			if (updateAvailable) this.counters.updated += 1;
			else this.counters.outOfSample += 1;
			this.#push({
				at: Date.now(),
				device: shortId(deviceId),
				kind: "checkin",
				status: 200,
				note: `update_available=${String(updateAvailable)}`,
				tone: updateAvailable ? "success" : "neutral",
			});
		} catch (error) {
			this.#noteRequestError("checkin", deviceId, error);
			return;
		}

		if (!updateAvailable) return;

		// 3. Отчёт: исход тянется из сеяного генератора, доля - вход симуляции.
		const result = pickBatchResult(rng, config.successSharePercent);
		const body: ReportInput = {
			campaign_id: config.campaignId,
			stage_id: stageId,
			result,
			...(result === "failure" ? { error_message: "симулированный сбой обновления" } : {}),
		};
		try {
			await reportUpdateAttempt(deviceId, body, { signal });
			if (result === "success") this.counters.success += 1;
			else this.counters.failure += 1;
			this.#push({
				at: Date.now(),
				device: shortId(deviceId),
				kind: "report",
				status: 200,
				note: result,
				tone: result === "success" ? "success" : "danger",
			});
		} catch (error) {
			// 503: попытка в Postgres, но событие не ушло в Kafka (§5.3).
			if (error instanceof ApiError && error.status === 503) {
				this.counters.failure += 1;
				this.#push({
					at: Date.now(),
					device: shortId(deviceId),
					kind: "report",
					status: 503,
					note: "отчёт записан в базу, но событие не ушло в Kafka: метрики стадии не изменятся",
					tone: "danger",
				});
				return;
			}
			this.#noteRequestError("report", deviceId, error);
		}
	}

	#noteRequestError(kind: LogLine["kind"], deviceId: string | undefined, error: unknown): void {
		if (error instanceof ApiError && error.status === 0) {
			// Запрос сорван остановкой прогона: это не сбой сервера.
			if (this.#controller?.signal.aborted) return;
		}
		this.counters.requestError += 1;
		const status = error instanceof ApiError ? error.status : undefined;
		const message =
			error instanceof ApiError ? (error.serverMessage ?? error.headline) : String(error);
		this.#push({
			at: Date.now(),
			device: deviceId ? shortId(deviceId) : "-",
			kind,
			status,
			note: message,
			tone: "danger",
		});
	}
}

export const batchRunner = new BatchRunner();
