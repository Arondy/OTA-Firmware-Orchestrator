/**
 * Карты статусов (00-CONTEXT §8.2, §9).
 *
 * Семантика цвета зафиксирована спецификацией и здесь собрана в одном месте:
 * янтарь (`progress`) означает «идёт прямо сейчас», акцент - основное действие,
 * три семантических оттенка - исход, нейтраль - инертное или неизвестное.
 * Новый статус в перечислении ломает компиляцию, потому что карты типизированы
 * как `Record<Enum, Meta>`.
 *
 * Статус никогда не кодируется цветом одним: всегда точка или иконка плюс
 * текстовая подпись (§8.2). Подпись набирается `fg-secondary` (>= 6:1 на
 * bg-base в обеих темах), а оттенок несёт точка: сами семантические цвета в
 * светлой теме на кегле 12-13px не дотягивают до 4.5:1, что проверено и
 * зафиксировано в docs/DECISIONS.md.
 */
// Иконки импортируются точечно, а не из баррела `phosphor-svelte`: в барреле
// 6050 модулей, и его предварительная сборка Vite не нужна ни в dev, ни в бандле.
// Экспорты без суффикса `Icon` в phosphor-svelte 3 помечены как deprecated.
import ArrowUUpLeftIcon from "phosphor-svelte/lib/ArrowUUpLeftIcon";
import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon";
import CircleDashedIcon from "phosphor-svelte/lib/CircleDashedIcon";
import PauseCircleIcon from "phosphor-svelte/lib/PauseCircleIcon";
import PlayCircleIcon from "phosphor-svelte/lib/PlayCircleIcon";
import TimerIcon from "phosphor-svelte/lib/TimerIcon";
import XCircleIcon from "phosphor-svelte/lib/XCircleIcon";
import type { Component } from "svelte";
import type { AttemptResult, CampaignStatus, DeviceStatus, StageStatus } from "$lib/api/types";

/** Точный тип иконки phosphor: все иконки пакета имеют одинаковую сигнатуру. */
export type StatusIcon = typeof CheckCircleIcon;

export type Tone = "accent" | "progress" | "success" | "held" | "danger" | "neutral";

/** Классы утилит на каждый тон. Значения полные строки, чтобы Tailwind их увидел. */
export interface ToneClasses {
	/** Фон точки-индикатора. */
	dot: string;
	/** Цвет иконки: используется на размерах от 18px, где достаточно 3:1. */
	icon: string;
	/** Цвет крупного числового показателя. */
	metric: string;
}

export const TONE_CLASSES: Record<Tone, ToneClasses> = {
	accent: {
		dot: "bg-accent",
		icon: "text-accent-text",
		metric: "text-accent-text",
	},
	progress: {
		dot: "bg-state-progress",
		icon: "text-state-progress",
		metric: "text-state-progress",
	},
	success: {
		dot: "bg-state-success",
		icon: "text-state-success",
		metric: "text-state-success",
	},
	held: {
		dot: "bg-state-held",
		icon: "text-state-held",
		metric: "text-state-held",
	},
	danger: {
		dot: "bg-state-danger",
		icon: "text-state-danger",
		metric: "text-state-danger",
	},
	neutral: {
		dot: "bg-state-neutral",
		icon: "text-fg-muted",
		metric: "text-fg-primary",
	},
};

/** Имя CSS-переменной тона: визуализации красятся через var(), без hex. */
export const TONE_CSS_VAR: Record<Tone, string> = {
	accent: "var(--accent-fill)",
	progress: "var(--state-progress)",
	success: "var(--state-success)",
	held: "var(--state-held)",
	danger: "var(--state-danger)",
	neutral: "var(--state-neutral)",
};

export interface StatusMeta {
	/** Подпись из словаря §9; синонимы не изобретаются. */
	label: string;
	tone: Tone;
	icon: Component;
	/** Точка пульсирует: состояние меняется без участия оператора (§8.6). */
	live?: boolean;
}

export const CAMPAIGN_STATUS: Record<CampaignStatus, StatusMeta> = {
	draft: { label: "черновик", tone: "neutral", icon: CircleDashedIcon },
	running: { label: "выполняется", tone: "progress", icon: PlayCircleIcon, live: true },
	paused: { label: "на паузе", tone: "held", icon: PauseCircleIcon },
	completed: { label: "завершена", tone: "success", icon: CheckCircleIcon },
	rolled_back: { label: "откачена", tone: "danger", icon: ArrowUUpLeftIcon },
};

export const STAGE_STATUS: Record<StageStatus, StatusMeta> = {
	pending: { label: "ожидает", tone: "neutral", icon: CircleDashedIcon },
	active: { label: "активна", tone: "progress", icon: PlayCircleIcon, live: true },
	passed: { label: "пройдена", tone: "success", icon: CheckCircleIcon },
	failed: { label: "провалена", tone: "danger", icon: XCircleIcon },
};

export const ATTEMPT_RESULT: Record<AttemptResult, StatusMeta> = {
	success: { label: "успех", tone: "success", icon: CheckCircleIcon },
	failure: { label: "ошибка", tone: "danger", icon: XCircleIcon },
	timeout: { label: "таймаут", tone: "danger", icon: TimerIcon },
};

export const DEVICE_STATUS: Record<DeviceStatus, StatusMeta> = {
	active: { label: "активно", tone: "success", icon: CheckCircleIcon },
	decommissioned: { label: "выведено из эксплуатации", tone: "neutral", icon: XCircleIcon },
};

export function campaignStatusMeta(status: CampaignStatus): StatusMeta {
	return CAMPAIGN_STATUS[status];
}

export function stageStatusMeta(status: StageStatus): StatusMeta {
	return STAGE_STATUS[status];
}

export function attemptResultMeta(result: AttemptResult): StatusMeta {
	return ATTEMPT_RESULT[result];
}

export function deviceStatusMeta(status: DeviceStatus): StatusMeta {
	return DEVICE_STATUS[status];
}

/*
 * Связность устройства - производный сигнал, а не новый оттенок (§8.2).
 *
 * `last_seen` в ответе накладывается из Redis с TTL 24 часа поверх Postgres,
 * а колонка в Postgres не обновляется никогда. Поэтому отсутствие `last_seen`
 * означает «не выходило на связь в последние сутки ИЛИ не отмечалось никогда»,
 * и писать «никогда» нельзя (§4, §9).
 */

/** Порог, после которого устройство считается «на связи». */
export const DEVICE_ONLINE_WINDOW_MS = 5 * 60 * 1000;

export const NO_LAST_SEEN_LABEL = "нет отметки";

export const NO_LAST_SEEN_TITLE =
	"Отметка хранится 24 часа. Устройство не выходило на связь в последние сутки или никогда не отмечалось.";

export interface DeviceLiveness {
	tone: Tone;
	label: string;
	title: string;
}

/**
 * Разбирает связность устройства. Принимает `last_seen` и `status` отдельно,
 * чтобы функцию можно было тестировать без полного объекта устройства.
 */
export function deviceLiveness(
	status: DeviceStatus,
	lastSeen: string | undefined,
	now: number,
): DeviceLiveness {
	if (status === "decommissioned") {
		return {
			tone: "neutral",
			label: "выведено из эксплуатации",
			title: "Устройство выведено из эксплуатации и не участвует в раскатках.",
		};
	}

	if (!lastSeen) {
		return { tone: "neutral", label: NO_LAST_SEEN_LABEL, title: NO_LAST_SEEN_TITLE };
	}

	const seenAt = Date.parse(lastSeen);
	if (Number.isNaN(seenAt)) {
		// Сервер отдал дату, которую браузер не смог разобрать: честнее сказать
		// «неизвестно», чем показать выдуманное время.
		return {
			tone: "neutral",
			label: "отметка не разобрана",
			title: `Значение отметки не удалось разобрать: ${lastSeen}`,
		};
	}

	const age = now - seenAt;
	if (age <= DEVICE_ONLINE_WINDOW_MS) {
		return {
			tone: "success",
			label: "на связи",
			title: `Последняя отметка: ${new Date(seenAt).toISOString()}`,
		};
	}

	return {
		tone: "neutral",
		label: "было на связи",
		title: `Последняя отметка: ${new Date(seenAt).toISOString()}`,
	};
}
