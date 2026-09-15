/**
 * Здоровье стека: опрос `GET /healthz` раз в 15 секунд с замером реальной
 * задержки в браузере (00-CONTEXT §5.4, §7.2).
 *
 * Задержка измеряется, а не выдумывается: это время последнего кругового
 * запроса `/healthz` от браузера до оркестратора и обратно. В подписи на экране
 * так и пишется - «задержка /healthz из браузера».
 *
 * Уведомления строго дозированы (§10.6): один тост на переход в состояние
 * «недоступен» и один на восстановление. Между ними - тишина, иначе поллинг
 * засыпал бы оператора сообщениями.
 */
import { createResource, type Resource, type ResourceSettled } from "./resource.svelte";
import { checkHealth } from "$lib/api/endpoints";
import { toast } from "./toast";

export const HEALTH_INTERVAL_MS = 15_000;
export const HEALTH_SAMPLES = 20;

const TOAST_KEY = "health";

export type HealthStatus = "ok" | "down" | "unknown";

export interface HealthSample {
	/** Дословное значение поля `status` из ответа `/healthz`. */
	reportedStatus: string;
	/** Круговое время запроса в миллисекундах, замерено в браузере. */
	latencyMs: number;
	/** Момент успешной проверки. */
	checkedAt: number;
}

class HealthMonitor {
	/** Последние успешные замеры задержки, не больше `HEALTH_SAMPLES`. */
	checks = $state<number[]>([]);
	lastCheckedAt = $state<number | undefined>(undefined);

	#notified: HealthStatus = "unknown";

	#resource: Resource<HealthSample> = createResource<HealthSample>(
		"healthz",
		async (signal) => {
			const startedAt = performance.now();
			const response = await checkHealth({ signal });
			return {
				reportedStatus: response.status,
				latencyMs: Math.round(performance.now() - startedAt),
				checkedAt: Date.now(),
			};
		},
		{
			intervalMs: HEALTH_INTERVAL_MS,
			onSettled: (settled) => this.#onSettled(settled),
		},
	);

	#onSettled(settled: ResourceSettled<HealthSample>): void {
		if (settled.ok && settled.data) {
			this.lastCheckedAt = settled.data.checkedAt;
			this.checks = [...this.checks, settled.data.latencyMs].slice(-HEALTH_SAMPLES);
		}
		this.#announce(this.status);
	}

	#announce(status: HealthStatus): void {
		if (status === this.#notified) return;

		const previous = this.#notified;
		this.#notified = status;

		if (status === "unknown") return;

		if (status === "down") {
			const error = this.error;
			if (error) toast.apiError(error, TOAST_KEY);
			return;
		}

		toast.dismiss(TOAST_KEY);
		if (previous === "down") toast.info("Связь с оркестратором восстановлена");
	}

	get status(): HealthStatus {
		if (this.#resource.error) return "down";
		if (this.#resource.data) return "ok";
		return "unknown";
	}

	/** Задержка последнего успешного запроса. При сбое связи не обновляется. */
	get latencyMs(): number | undefined {
		return this.#resource.data?.latencyMs;
	}

	get error() {
		return this.#resource.error;
	}

	async refresh(): Promise<void> {
		await this.#resource.refresh();
	}
}

/**
 * Единственный монитор здоровья на приложение: его держит AppShell, экраны
 * читают готовое состояние и не заводят собственных опросов `/healthz`.
 */
export const health = new HealthMonitor();
