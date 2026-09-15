/**
 * Здоровье rollout-контроллера: опрос его собственного Connect-эндпоинта
 * `POST /health.v1.HealthService/CheckHealth` раз в 15 секунд с замером
 * реальной задержки в браузере - так же, как монитор `/healthz` оркестратора.
 *
 * Путь проксируется тем же origin (`/controller/*`, см. Caddyfile и
 * vite.config.ts), поэтому CSP `connect-src 'self'` не нарушается.
 *
 * Сбой контроллера не поднимает глобальный баннер «API недоступен» и тосты:
 * баннер принадлежит оркестратору, через который идут все данные экрана,
 * а состояние контроллера показывает индикатор в верхней полосе.
 */
import { createResource, type Resource } from "./resource.svelte";
import { checkControllerHealth } from "$lib/api/endpoints";

export const CONTROLLER_HEALTH_INTERVAL_MS = 15_000;

export type ControllerHealthStatus = "ok" | "down" | "unknown";

interface ControllerHealthSample {
	latencyMs: number;
	checkedAt: number;
}

class ControllerHealthMonitor {
	#resource: Resource<ControllerHealthSample> = createResource<ControllerHealthSample>(
		"controller-health",
		async (signal) => {
			const startedAt = performance.now();
			await checkControllerHealth({ signal });
			return {
				latencyMs: Math.round(performance.now() - startedAt),
				checkedAt: Date.now(),
			};
		},
		{
			intervalMs: CONTROLLER_HEALTH_INTERVAL_MS,
			notifyAvailability: false,
		},
	);

	get status(): ControllerHealthStatus {
		if (this.#resource.error) return "down";
		if (this.#resource.data) return "ok";
		return "unknown";
	}

	/** Задержка последнего успешного запроса. При сбое связи не обновляется. */
	get latencyMs(): number | undefined {
		return this.#resource.data?.latencyMs;
	}

	get lastCheckedAt(): number | undefined {
		return this.#resource.data?.checkedAt;
	}

	async refresh(): Promise<void> {
		await this.#resource.refresh();
	}
}

/** Единственный монитор контроллера на приложение. */
export const controllerHealth = new ControllerHealthMonitor();
