/**
 * HTTP-клиент оркестратора.
 *
 * База - пустая строка: приложение ходит на относительные пути своего origin,
 * а адрес оркестратора подставляет прокси (Vite в dev, Caddy в проде). Поэтому
 * `API_BASE` - единственная константа, которую придётся менять при смене прокси.
 */
import { ApiError, parseErrorBody, type StatusText } from "./errors";

export const API_BASE = "";

/** Общий таймаут запроса. Совмещается с `AbortSignal` вызывающего кода. */
export const REQUEST_TIMEOUT_MS = 10_000;

export type HttpMethod = "GET" | "POST";

export type QueryValue = string | number | boolean | null | undefined;
export type Query = Record<string, QueryValue>;

export interface RequestOptions {
	query?: Query;
	body?: unknown;
	signal?: AbortSignal;
	statusText?: Partial<Record<number, StatusText>>;
}

interface RawResponse {
	status: number;
	body: unknown;
	requestId: string | undefined;
	url: string;
}

/** Пустой фильтр не должен уезжать на сервер как `?status=`: сервер вернёт 400. */
export function buildQuery(query: Query | undefined): string {
	if (!query) return "";

	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value === undefined || value === null) continue;
		if (typeof value === "string" && value.length === 0) continue;
		if (typeof value === "number" && Number.isNaN(value)) continue;
		params.set(key, String(value));
	}

	const encoded = params.toString();
	return encoded.length > 0 ? `?${encoded}` : "";
}

interface LinkedSignals {
	signal: AbortSignal;
	dispose: () => void;
}

/**
 * Совмещает таймаут с сигналом вызывающего кода.
 *
 * `AbortSignal.any` здесь намеренно не используется: он есть не во всех целевых
 * браузерах, а поведение при отмене нужно различать (отмена поллинга ошибкой не
 * считается, таймаут считается), поэтому контроллер собирается вручную.
 */
function linkSignals(caller: AbortSignal | undefined, timeoutMs: number): LinkedSignals {
	const controller = new AbortController();

	const timer = setTimeout(() => {
		controller.abort(new DOMException(`истекло время ожидания: ${timeoutMs} мс`, "TimeoutError"));
	}, timeoutMs);

	const onCallerAbort = () => {
		controller.abort(caller?.reason);
	};

	if (caller) {
		caller.addEventListener("abort", onCallerAbort, { once: true });
		if (caller.aborted) controller.abort(caller.reason);
	}

	return {
		signal: controller.signal,
		dispose: () => {
			clearTimeout(timer);
			caller?.removeEventListener("abort", onCallerAbort);
		},
	};
}

async function readBody(response: Response): Promise<unknown> {
	const text = await response.text();
	if (text.length === 0) return undefined;

	try {
		return JSON.parse(text) as unknown;
	} catch {
		// Не-JSON тело: отдаём строкой, `parseErrorBody` её подхватит.
		return text;
	}
}

function toApiError(
	response: { status: number },
	method: HttpMethod,
	url: string,
	requestId: string | undefined,
	body: unknown,
	statusText: Partial<Record<number, StatusText>>,
): ApiError {
	const parsed = parseErrorBody(body);
	return new ApiError({
		status: response.status,
		method,
		url,
		requestId,
		body,
		serverMessage: parsed.serverMessage,
		fields: parsed.fields,
		params: parsed.params,
		text: statusText[response.status],
	});
}

/**
 * Низкоуровневый запрос: бросает `ApiError` на не-2xx и на отсутствии ответа,
 * но не проверяет наличие тела - это делает `request<T>`.
 */
async function sendRaw(
	method: HttpMethod,
	path: string,
	options: RequestOptions = {},
): Promise<RawResponse> {
	const url = `${API_BASE}${path}${buildQuery(options.query)}`;
	const linked = linkSignals(options.signal, REQUEST_TIMEOUT_MS);
	const statusText = options.statusText ?? {};

	let response: Response;
	try {
		response = await fetch(url, {
			method,
			signal: linked.signal,
			headers: {
				Accept: "application/json",
				...(options.body === undefined ? {} : { "Content-Type": "application/json" }),
			},
			body: options.body === undefined ? undefined : JSON.stringify(options.body),
		});
	} catch (error) {
		linked.dispose();
		// Отмена инициатором (следующий тик поллинга) - штатная ситуация, не сбой.
		if (options.signal?.aborted) throw error;

		throw new ApiError({ status: 0, method, url, cause: error });
	}

	try {
		const requestId = response.headers.get("x-request-id") ?? undefined;
		const body = await readBody(response);

		if (!response.ok) {
			throw toApiError(response, method, url, requestId, body, statusText);
		}

		return { status: response.status, body, requestId, url };
	} finally {
		linked.dispose();
	}
}

/**
 * Запрос, который обязан вернуть тело. Пустое тело на успешном коде считается
 * ошибкой: лучше явный сбой, чем `undefined`, молча протёкший в интерфейс.
 */
export async function request<T>(
	method: HttpMethod,
	path: string,
	options: RequestOptions = {},
): Promise<T> {
	const { status, body, requestId, url } = await sendRaw(method, path, options);

	if (body === undefined || body === null) {
		throw new ApiError({
			status,
			method,
			url,
			requestId,
			serverMessage: "empty response body",
			text: options.statusText?.[status],
		});
	}

	return body as T;
}

/**
 * Запрос без тела в ответе. Используется для `POST /campaigns/{id}/rollback`:
 * сервер отвечает `202` с пустым телом, вызывать `res.json()` там нельзя (§5.3).
 */
export async function requestVoid(
	method: HttpMethod,
	path: string,
	options: RequestOptions = {},
): Promise<void> {
	await sendRaw(method, path, options);
}
