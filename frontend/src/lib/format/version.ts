/**
 * Отображение версий. Разбор, сравнение и сортировка semver живут в
 * `logic/semver.ts` и проброшены сюда; дублирования реализации нет.
 */
export { compareSemver, isSemver, parseSemver, sortSemver } from "$lib/logic/semver";
export type { SemVer } from "$lib/logic/semver";

import { isSemver } from "$lib/logic/semver";

export const UNPARSEABLE_VERSION_LABEL = "версия не разобрана";

/** Контракт API не допускает префикс `v` и сокращённые формы, поэтому нормализация сводится к проверке. */
export function formatVersion(version: string | undefined): string {
	if (!version) return "";
	return isSemver(version) ? version : UNPARSEABLE_VERSION_LABEL;
}
