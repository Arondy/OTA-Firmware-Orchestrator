/**
 * Машина состояний кампании (00-CONTEXT §4).
 *
 * Чистая функция без обращений к API: интерфейс обязан запрещать невозможные
 * действия до отправки запроса, а не показывать ошибку сервера постфактум.
 *
 *   draft      --start-->    running
 *   running    --pause-->    paused
 *   paused     --resume-->   running
 *   running    --advance-->  running | completed   (автоматически, контроллером)
 *   running    --rollback--> rolled_back           (автоматически, контроллером)
 *   paused     --rollback--> rolled_back           (вручную, асинхронно, 202)
 *   completed, rolled_back - терминальные, действий нет
 */
import type { CampaignStatus } from "$lib/api/types";
import type { Tone } from "./status";

export type CampaignAction = "start" | "pause" | "resume" | "rollback";

const TRANSITIONS: Record<CampaignStatus, CampaignAction[]> = {
	draft: ["start"],
	running: ["pause", "rollback"],
	paused: ["resume", "rollback"],
	completed: [],
	rolled_back: [],
};

/** Действия, доступные из данного статуса. Порядок - порядок кнопок в интерфейсе. */
export function allowedActions(status: CampaignStatus): CampaignAction[] {
	return TRANSITIONS[status];
}

/** Терминальный статус: ручных действий нет и поллинг не нужен. */
export function isTerminal(status: CampaignStatus): boolean {
	return status === "completed" || status === "rolled_back";
}

/**
 * Статусы, при которых кампанию стоит опрашивать: её состояние может изменить
 * контроллер без участия оператора (продвижение стадии, автоматический откат).
 * Используется как предикат `enabled()` у ресурса (§7.2).
 */
export function isLive(status: CampaignStatus): boolean {
	return status === "running" || status === "paused";
}

export interface ActionMeta {
	/** Подпись на кнопке: повелительное наклонение, sentence case (§9). */
	label: string;
	/** Подпись на кнопке, пока запрос не завершился. */
	pendingLabel: string;
	tone: Tone;
	/** Требовать явного подтверждения перед отправкой. */
	confirm?: boolean;
	/**
	 * Действие асинхронное на стороне сервера: `202` с пустым телом, статус
	 * изменится позже. Оптимистично менять статус нельзя (§5.3).
	 */
	async?: boolean;
	/** Короткое пояснение для подтверждения и всплывающей подсказки. */
	description: string;
}

export const ACTION_META: Record<CampaignAction, ActionMeta> = {
	start: {
		label: "Запустить",
		pendingLabel: "Запуск",
		tone: "accent",
		description:
			"Кампания перейдёт из черновика в статус «выполняется», первая стадия станет активной.",
	},
	pause: {
		label: "Поставить на паузу",
		pendingLabel: "Пауза",
		tone: "held",
		description: "Устройства перестанут получать обновление, накопленные метрики сохранятся.",
	},
	resume: {
		label: "Возобновить",
		pendingLabel: "Возобновление",
		tone: "accent",
		description: "Раскатка продолжится с активной стадии.",
	},
	rollback: {
		label: "Откатить",
		pendingLabel: "Откат запрошен",
		tone: "danger",
		confirm: true,
		async: true,
		description:
			"Оркестратор примет запрос и передаст решение контроллеру. Статус «откачена» появится позже, когда решение применится: следите за статусом кампании.",
	},
};

export function actionMeta(action: CampaignAction): ActionMeta {
	return ACTION_META[action];
}

/**
 * Канонический текст подтверждения отката: один диалог на всех
 * экранах - detail, строки обзора и меню списка кампаний.
 */
export const ROLLBACK_CONFIRM = {
	title: "Откатить кампанию?",
	body: "Оркестратор передаст решение об откате контроллеру. Статус сменится на «откачена» асинхронно, обычно в течение нескольких секунд. Действие нельзя отменить.",
	confirm: "Откатить",
	cancel: "Отмена",
} as const;
