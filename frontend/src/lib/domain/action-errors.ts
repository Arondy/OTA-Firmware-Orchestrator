/**
 * Русские тексты сбоев действий кампании.
 *
 * Сервер отвечает кодом и английской строкой; оператор видит перевод,
 * а дословная строка остаётся в раскрывающейся отладке (§5.3). Тексты собраны
 * здесь, чтобы detail-экран, строки обзора и меню списка говорили одинаково.
 */
import type { ApiError } from "$lib/api/errors";
import type { CampaignAction } from "./transitions";

/**
 * Заголовок инлайн-ошибки действия. `statusLabel` - подпись текущего статуса
 * кампании из словаря: нужна там, где текст зависит от статуса
 * («Откат недоступен для кампании в статусе «завершена»»).
 */
export function actionErrorHeadline(
	action: CampaignAction,
	error: ApiError,
	statusLabel: string | undefined,
): string {
	if (error.status === 404) return "Кампания не найдена";

	if (error.status === 409) {
		// Единственный конфликт в контракте: другая живая кампания той же модели.
		return "Для этой модели уже выполняется другая кампания";
	}

	if (error.status === 400) {
		switch (action) {
			case "start":
				return "Запустить можно только кампанию в статусе «черновик»";
			case "pause":
				return "Пауза доступна только для выполняющейся кампании";
			case "resume":
				return "Возобновить можно только кампанию на паузе";
			case "rollback":
				return statusLabel
					? `Откат недоступен для кампании в статусе «${statusLabel}»`
					: "Откат недоступен для кампании в этом статусе";
		}
	}

	return error.headline;
}
