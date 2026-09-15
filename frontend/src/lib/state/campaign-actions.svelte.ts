/**
 * Действия над кампанией: единая машина для detail-экрана, строк обзора и
 * меню списка.
 *
 * Договорённости:
 *   - `start` / `pause` / `resume` синхронные (200 + полная кампания): ответ
 *     передаётся экранам через `onChanged`, чтобы деталь подставила его без
 *     лишнего запроса;
 *   - `rollback` асинхронный: 202 с пустым телом, затем опрос статуса каждые
 *     1500 мс не дольше 20 с. Оптимистичного статуса нет никогда: пока решение
 *     не применилось, экран показывает настоящий статус и полосу «Применяем
 *     откат…»;
 *   - таймаут опроса не объявляется сбоем: запрос был принят (202), статус
 *     ещё не изменился. Экран показывает информационный баннер с кнопкой
 *     «Обновить»;
 *   - ошибочные ответы сохраняются по кампании и показываются инлайн,
 *     дословная строка сервера остаётся в отладке (§5.3).
 */
import {
	getRolloutCampaign,
	pauseRolloutCampaign,
	resumeRolloutCampaign,
	rollbackRolloutCampaign,
	startRolloutCampaign,
} from "$lib/api/endpoints";
import { ApiError } from "$lib/api/errors";
import type { Campaign, CampaignStatus } from "$lib/api/types";
import { actionErrorHeadline } from "$lib/domain/action-errors";
import type { CampaignAction } from "$lib/domain/transitions";
import { mutation } from "./mutation.svelte";
import { toast } from "./toast";

const ROLLBACK_POLL_MS = 1500;
const ROLLBACK_WINDOW_MS = 20_000;

export interface ActionFailure {
	action: CampaignAction;
	error: ApiError;
}

type RollbackPhase = "inFlight" | "timedOut";

type ChangeListener = (updated?: Campaign) => Promise<void> | void;

class CampaignActions {
	#rollback = $state<Record<string, RollbackPhase>>({});
	#failures = $state<Record<string, ActionFailure | undefined>>({});
	#listeners = new Set<ChangeListener>();
	#timers = new Set<ReturnType<typeof setTimeout>>();

	/** Экраны подписываются: аргумент - кампания из синхронного ответа, если он был. */
	onChanged(listener: ChangeListener): () => void {
		this.#listeners.add(listener);
		return () => {
			this.#listeners.delete(listener);
		};
	}

	async #notify(updated?: Campaign): Promise<void> {
		for (const listener of [...this.#listeners]) await listener(updated);
	}

	isPending(id: string, action: CampaignAction): boolean {
		return mutation.isPending(`${id}:${action}`);
	}

	/** Откат запрошен (202) и ещё не применился: статус опрашивается каждые 1500 мс. */
	isRollbackInFlight(id: string): boolean {
		return this.#rollback[id] === "inFlight";
	}

	/** Окно ожидания вышло: запрос принят, статус ещё не изменился. */
	isRollbackTimedOut(id: string): boolean {
		return this.#rollback[id] === "timedOut";
	}

	/** Последняя ошибка действия по кампании; `undefined`, когда сбоев не было. */
	failure(id: string): ActionFailure | undefined {
		return this.#failures[id];
	}

	/**
	 * Экран сообщает загруженный статус: как только виден факт отката,
	 * ожидание снимается и поднимается единственный тост успеха.
	 */
	noteStatus(id: string, status: CampaignStatus): void {
		if (status !== "rolled_back" || this.#rollback[id] === undefined) return;
		this.#finishRollback(id);
	}

	async start(id: string, label: string, statusLabel?: string): Promise<void> {
		await this.#sync(id, "start", statusLabel, async () => {
			const campaign = await startRolloutCampaign(id);
			toast.success(`${label}: кампания запущена`, undefined, `${id}:start`);
			return campaign;
		});
	}

	async pause(id: string, label: string, statusLabel?: string): Promise<void> {
		await this.#sync(id, "pause", statusLabel, async () => {
			const campaign = await pauseRolloutCampaign(id);
			toast.success(`${label}: кампания на паузе`, undefined, `${id}:pause`);
			return campaign;
		});
	}

	async resume(id: string, label: string, statusLabel?: string): Promise<void> {
		await this.#sync(id, "resume", statusLabel, async () => {
			const campaign = await resumeRolloutCampaign(id);
			toast.success(`${label}: кампания возобновлена`, undefined, `${id}:resume`);
			return campaign;
		});
	}

	async rollback(id: string, label: string, statusLabel?: string): Promise<void> {
		this.#failures[id] = undefined;
		try {
			await mutation.run(
				`${id}:rollback`,
				async () => {
					await rollbackRolloutCampaign(id);
					this.#rollback[id] = "inFlight";
					toast.info(`${label}: откат запрошен`);
					await this.#notify();
					this.#watchRollback(id, label, Date.now());
				},
				{ notify: false },
			);
		} catch (error) {
			this.#noteFailure(id, "rollback", error, statusLabel);
		}
	}

	/** Синхронное действие: ответ уходит экранам сразу, без повторного запроса. */
	async #sync(
		id: string,
		action: CampaignAction,
		statusLabel: string | undefined,
		call: () => Promise<Campaign>,
	): Promise<void> {
		this.#failures[id] = undefined;
		try {
			await mutation.run(
				`${id}:${action}`,
				async () => {
					const campaign = await call();
					await this.#notify(campaign);
				},
				{ notify: false },
			);
		} catch (error) {
			this.#noteFailure(id, action, error, statusLabel);
		}
	}

	#noteFailure(
		id: string,
		action: CampaignAction,
		error: unknown,
		statusLabel: string | undefined,
	): void {
		if (!(error instanceof ApiError)) return;

		this.#failures[id] = { action, error };
		const parts = [error.hint];
		if (error.serverMessage) parts.push(`Ответ сервера: ${error.serverMessage}`);
		toast.error(
			actionErrorHeadline(action, error, statusLabel),
			parts.join(" "),
			`${id}:${action}`,
		);
		void this.#notify();
	}

	#watchRollback(id: string, label: string, startedAt: number): void {
		const timer = setTimeout(() => {
			this.#timers.delete(timer);
			void this.#pollRollback(id, label, startedAt);
		}, ROLLBACK_POLL_MS);
		this.#timers.add(timer);
	}

	async #pollRollback(id: string, label: string, startedAt: number): Promise<void> {
		if (this.#rollback[id] !== "inFlight") return;

		if (Date.now() - startedAt >= ROLLBACK_WINDOW_MS) {
			// Не сбой: 202 принят, решение применяется асинхронно.
			this.#rollback[id] = "timedOut";
			await this.#notify();
			return;
		}

		try {
			const campaign = await getRolloutCampaign(id);
			if (this.#rollback[id] !== "inFlight") return;
			if (campaign.status === "rolled_back") {
				this.#finishRollback(id);
				return;
			}
		} catch {
			// Оркестратор мог не ответить на саму проверку: пробуем следующим тиком.
		}

		this.#watchRollback(id, label, startedAt);
	}

	#finishRollback(id: string): void {
		delete this.#rollback[id];
		toast.success("Кампания откачена", undefined, `${id}:rollback`);
		void this.#notify();
	}

	/** Снять все ожидания отката: используется тестами между случаями. */
	cancelWatches(): void {
		for (const timer of this.#timers) clearTimeout(timer);
		this.#timers.clear();
		this.#rollback = {};
		this.#failures = {};
	}
}

export const campaignActions = new CampaignActions();
