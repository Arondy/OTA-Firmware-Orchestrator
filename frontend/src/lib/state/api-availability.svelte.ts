/**
 * Единственный флаг доступности API на всё приложение (00-CONTEXT §7.1).
 *
 * Флаг взводится ошибками с `status === 0`, то есть когда запрос вообще не дошёл
 * до сервера (сеть, DNS, таймаут), а также ответом `502`: его отдаёт прокси
 * контейнера фронтенда, когда оркестратор за ним не отвечает, - с точки зрения
 * браузера это та же недоступность. Ответ `500` или `404` означает, что сервер
 * жив и отвечает, - глобальный баннер в этом случае показывать нельзя.
 */
import type { ApiError } from "$lib/api/errors";

class ApiAvailability {
	/** `true`, когда последний известный исход запроса - «сервер не ответил». */
	down = $state(false);
	/** Момент последнего успешного ответа: для подписи «был доступен N назад». */
	lastSuccessAt = $state<number | undefined>(undefined);
	/** Момент последнего сбоя связи. */
	lastFailureAt = $state<number | undefined>(undefined);
	/** Дословная причина последнего сбоя связи для раскрывающейся отладки. */
	lastFailureDetail = $state<string | undefined>(undefined);

	noteSuccess(now: number): void {
		this.down = false;
		this.lastSuccessAt = now;
	}

	noteFailure(error: ApiError, now: number): void {
		if (!error.isNetworkFailure && error.status !== 502) {
			// Сервер ответил чем-то кроме 502, значит он доступен. Флаг не трогаем.
			return;
		}
		this.down = true;
		this.lastFailureAt = now;
		this.lastFailureDetail = error.debugLine;
	}

	/** Полный сброс: нужен юнит-тестам, чтобы состояние не перетекало между ними. */
	reset(): void {
		this.down = false;
		this.lastSuccessAt = undefined;
		this.lastFailureAt = undefined;
		this.lastFailureDetail = undefined;
	}
}

export const apiAvailability = new ApiAvailability();
