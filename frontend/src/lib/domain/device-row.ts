/**
 * Производные сигналы строки устройства.
 *
 * Маркер связности считается только от времени последней отметки и никогда не
 * называет устройство «офлайн»: отсутствие `last_seen` означает «нет отметки»
 * с пояснением про TTL Redis 24 часа (00-CONTEXT §9). Границы маркера:
 * свежая отметка (< 5 мин), рабочая (до часа), давняя (больше часа).
 */
import { DEVICE_ONLINE_WINDOW_MS } from "./status";

/** Часовая граница между «рабочей» и «давней» отметкой. */
export const LAST_SEEN_STALE_MS = 60 * 60 * 1000;

export type LastSeenMarker = "fresh" | "recent" | "stale";

/** Класс точки маркера: тон без подложки, смысл дублируется подсказкой. */
export const MARKER_DOT: Record<LastSeenMarker, string> = {
	fresh: "bg-state-success",
	recent: "bg-state-neutral",
	stale: "bg-fg-muted",
};

export const MARKER_TITLE = "оценка по времени последней отметки";

/**
 * Маркер по времени отметки; `undefined`, когда отметки нет вовсе: тогда
 * строка показывает «нет отметки», а не рисует точку.
 */
export function lastSeenMarker(
	lastSeen: string | undefined,
	now: number,
): LastSeenMarker | undefined {
	if (!lastSeen) return undefined;
	const seenAt = Date.parse(lastSeen);
	if (Number.isNaN(seenAt)) return undefined;

	const age = now - seenAt;
	if (age <= DEVICE_ONLINE_WINDOW_MS) return "fresh";
	if (age <= LAST_SEEN_STALE_MS) return "recent";
	return "stale";
}

/**
 * Устройство уже на целевой версии выполняющейся кампании своей модели:
 * версия сравнивается со строкой версии прошивки кампании, разрешённой через
 * реестр (`firmwareIndex`), а не с идентификатором.
 */
export function isOnTargetVersion(
	deviceModel: string,
	currentVersion: string,
	runningTargetVersionByModel: ReadonlyMap<string, string>,
): boolean {
	const target = runningTargetVersionByModel.get(deviceModel);
	return target !== undefined && target === currentVersion;
}

export const TARGET_CHIP_TITLE = "устройство уже на версии раскатки";
