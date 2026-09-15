/**
 * Единственное место, где HTTP-ответ превращается в русский текст для оператора.
 * Компоненты не смотрят на `res.status` и не знают про коды: они получают
 * уже переведённый `ApiError` (00-CONTEXT §7.3).
 *
 * Оригинальное сообщение сервера сохраняется в `serverMessage` и показывается
 * рядом с переводом - спека требует никогда его не проглатывать (§5.3).
 */

/** Разобранный ответ сервера об ошибке: `{ error }`, `{ error, fields }` или `{ error, params }`. */
export interface ServerErrorBody {
	error?: unknown;
	fields?: unknown;
	params?: unknown;
}

/** Русский текст для кодов, где таблица переводов неприменима. */
export interface StatusText {
	message: string;
	hint?: string;
}

export interface ApiErrorInit {
	/** HTTP-код. `0` означает, что запрос не дошёл: сеть, DNS или таймаут. */
	status: number;
	method: string;
	url: string;
	/** Дословный текст из поля `error` ответа. */
	serverMessage?: string;
	/** Ошибки полей из `400 { error, fields }`; ключи - snake_case имена DTO. */
	fields?: Record<string, string>;
	/** Имена query-параметров из `400 { error, params }`. */
	params?: string[];
	/** Значение заголовка `X-Request-Id`, если сервер его отдал. */
	requestId?: string;
	/** Сырое тело ответа для раскрывающейся отладки. */
	body?: unknown;
	/** Причина, по которой запрос вообще не состоялся (сеть или таймаут). */
	cause?: unknown;
	/** Текст вместо таблицы переводов: для кодов вне контракта. */
	text?: StatusText;
}

/**
 * Перевод известных сообщений сервера. Ключи - дословные строки из
 * `ota-orchestrator`; значение `message` показывается как заголовок,
 * `hint` - как пояснение, когда причину можно устранить на месте.
 *
 * В строке про откат сервер делает опечатку (`non- running/paused` с лишним
 * пробелом). Ключ здесь приведён ровно так, как её отдаёт сервер, а перевод
 * опечатку не повторяет: дословно этот текст в интерфейсе не показывается.
 */
const SERVER_MESSAGE_MAP: Record<string, { message: string; hint?: string }> = {
	"device not found": {
		message: "Устройство не найдено",
		hint: "Идентификатор устарел или устройство удалено. Обновите список устройств.",
	},
	"firmware version not found": {
		message: "Версия прошивки не найдена",
		hint: "Запись удалена из реестра или идентификатор устарел. Обновите реестр прошивок.",
	},
	"rollout campaign not found": {
		message: "Кампания не найдена",
		hint: "Идентификатор устарел. Обновите список кампаний.",
	},
	"rollout stage not found in this campaign": {
		message: "Стадия не найдена в этой кампании",
		hint: "Стадия относится к другой кампании либо кампанию пересоздали.",
	},
	"this firmware version for this model already exists": {
		message: "Такая версия прошивки для этой модели уже зарегистрирована",
		hint: "Пара «модель устройства + версия прошивки» уникальна. Возьмите существующую запись из реестра.",
	},
	"another campaign for this model is already running": {
		message: "Для этой модели уже выполняется другая кампания",
		hint: "На одну модель устройства допускается только одна выполняющаяся кампания. Завершите, приостановите или откатите текущую и повторите действие.",
	},
	"can't start non-draft rollout campaign": {
		message: "Запустить можно только кампанию в статусе «черновик»",
		hint: "Кампания уже запущена, завершена или откачена. Обновите страницу кампании.",
	},
	"can't pause non-running rollout campaign": {
		message: "Пауза доступна только для выполняющейся кампании",
		hint: "Кампания не в статусе «выполняется». Обновите страницу кампании.",
	},
	"can't resume non-paused rollout campaign": {
		message: "Возобновить можно только кампанию на паузе",
		hint: "Кампания не в статусе «на паузе». Обновите страницу кампании.",
	},
	"can't rollback non- running/paused rollout campaign": {
		message: "Откат недоступен для кампании в этом статусе",
		hint: "Откат возможен только для выполняющейся кампании или кампании на паузе.",
	},
	"rollout stage for this rollout campaign with such index already exists": {
		message: "Стадия с таким порядковым номером уже есть",
		hint: "Номера стадий начинаются с 0 и не должны повторяться. Проверьте порядок стадий в форме.",
	},
	"wrong campaign status": {
		message: "Статус кампании не допускает этого действия",
		hint: "Обновите страницу кампании: статус мог измениться решением контроллера.",
	},
	"wrong stage status": {
		message: "Статус стадии не допускает этого действия",
		hint: "Обновите страницу кампании: стадия могла перейти в другое состояние.",
	},
	"this device's model is not updated in this campaign": {
		message: "Модель этого устройства не обновляется в данной кампании",
		hint: "Выберите кампанию, у которой модель устройства совпадает с моделью этого устройства.",
	},
	"failed to produce update result": {
		message: "Не удалось опубликовать результат обновления",
		hint: "Оркестратор не смог записать событие в Kafka. Повторите отправку отчёта.",
	},
	"request didn't pass validation": {
		message: "Запрос не прошёл проверку",
		hint: "Исправьте подсвеченные поля и отправьте форму снова.",
	},
	"invalid request query parameters": {
		message: "Недопустимые параметры списка",
		hint: "Проверьте значения фильтров, номера страницы и размера страницы.",
	},
	"invalid JSON": {
		message: "Сервер не смог разобрать тело запроса",
		hint: "Тело должно быть одним JSON-объектом без лишних символов.",
	},
	"internal server error": {
		message: "Внутренняя ошибка сервера",
		hint: "Повторите действие. Если ошибка повторяется, сообщите идентификатор запроса и посмотрите логи оркестратора.",
	},
	"service unavailable": {
		message: "Сервис недоступен",
		hint: "Зависимость оркестратора не отвечает. Повторите действие позже.",
	},
	// Не сообщение сервера: так клиент помечает успешный код с пустым телом там,
	// где тело обязательно.
	"empty response body": {
		message: "Сервер вернул пустой ответ",
		hint: "Ожидалось тело ответа. Повторите запрос и сообщите идентификатор запроса.",
	},
};

/** Ответ по умолчанию для каждого известного класса кодов. */
const STATUS_FALLBACK: Record<number, { message: string; hint?: string }> = {
	0: {
		message: "Сервер недоступен или истекло время ожидания",
	},
	400: { message: "Запрос не прошёл проверку", hint: "Проверьте введённые значения." },
	404: { message: "Данные не найдены", hint: "Запись удалена или идентификатор устарел." },
	409: {
		message: "Конфликт с текущим состоянием",
		hint: "Данные изменились. Обновите экран и повторите действие.",
	},
	413: {
		message: "Тело запроса превышает лимит 1 МиБ",
		hint: "Уменьшите объём отправляемых данных.",
	},
	500: {
		message: "Внутренняя ошибка сервера",
		hint: "Повторите действие и сообщите идентификатор запроса.",
	},
	// 502 отдаёт прокси контейнера фронтенда (Caddy), когда оркестратор не
	// отвечает. Для оператора это тот же исход, что и «нет ответа», поэтому
	// сообщение совпадает со статусом 0.
	502: {
		message: "Сервер недоступен или истекло время ожидания",
	},
	503: {
		message: "Сервис недоступен",
		hint: "Зависимость оркестратора не отвечает. Повторите действие позже.",
	},
};

const UNKNOWN_FIELD_PREFIX = "json: unknown field in request body:";
const BODY_LIMIT_PREFIX = "request body exceeds limit in";

function resolveText(status: number, serverMessage: string | undefined): StatusText {
	const raw = serverMessage?.trim();

	if (raw) {
		const exact = SERVER_MESSAGE_MAP[raw];
		if (exact) return exact;

		// `json: unknown field in request body: <key>` - сообщение с динамическим хвостом.
		if (raw.startsWith(UNKNOWN_FIELD_PREFIX)) {
			const field = raw.slice(UNKNOWN_FIELD_PREFIX.length).trim();
			return {
				message: "Сервер отклонил неизвестное поле в теле запроса",
				hint: `Поле «${field}» отсутствует в контракте. Клиент отправил лишнее поле.`,
			};
		}

		// `request body exceeds limit in 1048576 bytes` - тоже с динамическим хвостом.
		if (raw.startsWith(BODY_LIMIT_PREFIX)) {
			return {
				message: "Тело запроса превышает лимит",
				hint: "Оркестратор принимает не больше 1 МиБ в одном запросе.",
			};
		}
	}

	return (
		STATUS_FALLBACK[status] ?? {
			message: `Непредвиденный ответ сервера (код ${status})`,
			hint: "Повторите действие и сообщите идентификатор запроса.",
		}
	);
}

/**
 * Ошибка запроса к API. Всегда несёт и русский текст для оператора, и дословное
 * сообщение сервера, и идентификатор запроса для отладки.
 */
export class ApiError extends Error {
	readonly status: number;
	readonly method: string;
	readonly url: string;
	readonly serverMessage: string | undefined;
	readonly fields: Record<string, string> | undefined;
	readonly params: string[] | undefined;
	readonly requestId: string | undefined;
	readonly body: unknown;
	/** Русский заголовок: что именно не получилось. */
	readonly headline: string;
	/** Русское пояснение: что можно сделать. */
	readonly hint: string | undefined;

	constructor(init: ApiErrorInit) {
		const resolved = init.text ?? resolveText(init.status, init.serverMessage);
		super(resolved.message, { cause: init.cause });

		this.name = "ApiError";
		this.status = init.status;
		this.method = init.method;
		this.url = init.url;
		this.serverMessage = init.serverMessage;
		this.fields = init.fields;
		this.params = init.params;
		this.requestId = init.requestId;
		this.body = init.body;
		this.headline = resolved.message;
		this.hint = resolved.hint;

		// `message` совпадает с заголовком, чтобы `String(error)` в логах был осмысленным.
		this.message = resolved.message;
	}

	/** Запрос не дошёл до сервера: сеть, DNS или таймаут. */
	get isNetworkFailure(): boolean {
		return this.status === 0;
	}

	/** Короткое описание для раскрывающейся отладки. */
	get debugLine(): string {
		return `${this.method} ${this.url} -> ${this.status === 0 ? "нет ответа" : this.status}`;
	}
}

/**
 * Защита от мусора в теле ошибки: сервер может отдать не JSON, пустое тело
 * или JSON неожиданной формы. Ничего из этого не должно ронять клиент.
 */
export function parseErrorBody(body: unknown): {
	serverMessage: string | undefined;
	fields: Record<string, string> | undefined;
	params: string[] | undefined;
} {
	if (typeof body !== "object" || body === null) {
		return {
			serverMessage: typeof body === "string" && body.length > 0 ? body : undefined,
			fields: undefined,
			params: undefined,
		};
	}

	const candidate = body as ServerErrorBody;

	const serverMessage =
		typeof candidate.error === "string" && candidate.error.length > 0 ? candidate.error : undefined;

	let fields: Record<string, string> | undefined;
	if (typeof candidate.fields === "object" && candidate.fields !== null) {
		const entries = Object.entries(candidate.fields as Record<string, unknown>).filter(
			(entry): entry is [string, string] => typeof entry[1] === "string",
		);
		if (entries.length > 0) fields = Object.fromEntries(entries);
	}

	let params: string[] | undefined;
	if (Array.isArray(candidate.params)) {
		const values = candidate.params.filter((item): item is string => typeof item === "string");
		if (values.length > 0) params = values;
	}

	return { serverMessage, fields, params };
}
