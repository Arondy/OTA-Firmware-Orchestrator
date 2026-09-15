import { beforeEach, describe, expect, it } from "vitest";
import { ApiError } from "$lib/api/errors";
import { apiAvailability } from "./api-availability.svelte";

/**
 * Флаг доступности API: сеть и 502 от прокси считаются недоступностью,
 * живые ответы сервера - нет (00-CONTEXT §7.1: оркестратор за Caddy может
 * отвечать 502, и это тот же исход, что «нет ответа»).
 */

function errorOf(status: number): ApiError {
	return new ApiError({ status, method: "GET", url: "/api/v1/test" });
}

describe("apiAvailability", () => {
	beforeEach(() => {
		apiAvailability.reset();
	});

	it("взводит флаг, когда запрос не дошёл до сервера", () => {
		apiAvailability.noteFailure(errorOf(0), Date.now());
		expect(apiAvailability.down).toBe(true);
	});

	it("взводит флаг на 502 от прокси", () => {
		apiAvailability.noteFailure(errorOf(502), Date.now());
		expect(apiAvailability.down).toBe(true);
	});

	it("502 сообщает то же, что и отсутствие ответа", () => {
		expect(errorOf(502).headline).toBe(errorOf(0).headline);
		expect(errorOf(502).headline).toBe("Сервер недоступен или истекло время ожидания");
	});

	it("не взводит флаг на живые ответы сервера", () => {
		for (const status of [400, 404, 409, 500, 503]) {
			apiAvailability.noteFailure(errorOf(status), Date.now());
			expect(apiAvailability.down).toBe(false);
		}
	});

	it("успешный ответ снимает флаг", () => {
		apiAvailability.noteFailure(errorOf(502), Date.now());
		expect(apiAvailability.down).toBe(true);
		apiAvailability.noteSuccess(Date.now());
		expect(apiAvailability.down).toBe(false);
	});
});
