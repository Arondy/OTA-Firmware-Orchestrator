import { afterEach, describe, expect, it, vi } from "vitest";
import { copyText, shortId, uuidHex } from "./id";

describe("shortId", () => {
	it("сокращает UUID до первых 8 и последних 5 цифр", () => {
		// Пример из 00-CONTEXT §9.
		expect(shortId("3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c")).toBe("3f6d2d8e\u2026a3b4c");
	});

	it("работает и с записью без дефисов", () => {
		expect(shortId("3f6d2d8e4b1c4a5f9a7b8e7d1c2a3b4c")).toBe("3f6d2d8e\u2026a3b4c");
	});

	it("возвращает пустую строку для undefined", () => {
		expect(shortId(undefined)).toBe("");
	});

	it("не режет строку, которая не является UUID", () => {
		expect(shortId("demo-sensor-v1")).toBe("demo-sensor-v1");
	});
});

describe("uuidHex", () => {
	it("убирает дефисы", () => {
		expect(uuidHex("3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c")).toBe(
			"3f6d2d8e4b1c4a5f9a7b8e7d1c2a3b4c",
		);
	});
});

describe("copyText", () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("пустую строку не копирует", async () => {
		expect(await copyText("")).toBe(false);
	});

	it("пользуется современным clipboard API, когда он доступен", async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal("navigator", { clipboard: { writeText } });

		expect(await copyText("3f6d2d8e")).toBe(true);
		expect(writeText).toHaveBeenCalledWith("3f6d2d8e");
	});

	it("при отказе clipboard API пробует запасной путь", async () => {
		vi.stubGlobal("navigator", {
			clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
		});

		const select = vi.fn();
		const remove = vi.fn();
		const appendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => {
			Object.assign(node, { select, remove });
			return node;
		});
		const execCommand = vi.fn().mockReturnValue(true);
		document.execCommand = execCommand;

		expect(await copyText("значение")).toBe(true);
		expect(execCommand).toHaveBeenCalledWith("copy");
		expect(select).toHaveBeenCalled();
		expect(remove).toHaveBeenCalled();

		appendChild.mockRestore();
	});

	it("честно сообщает о неудаче, когда оба пути не сработали", async () => {
		vi.stubGlobal("navigator", {});
		document.execCommand = vi.fn().mockReturnValue(false);

		expect(await copyText("значение")).toBe(false);
	});
});
