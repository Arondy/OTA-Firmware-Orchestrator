import { describe, expect, it } from "vitest";
import { UNPARSEABLE_VERSION_LABEL, formatVersion, isSemver } from "./version";

describe("formatVersion", () => {
	it("показывает валидную версию как есть", () => {
		expect(formatVersion("1.4.2")).toBe("1.4.2");
		expect(formatVersion("1.0.0-rc.1")).toBe("1.0.0-rc.1");
	});

	it("не нормализует то, чего контракт не допускает, а признаёт проблему", () => {
		expect(formatVersion("v1.4.2")).toBe(UNPARSEABLE_VERSION_LABEL);
		expect(formatVersion("latest")).toBe(UNPARSEABLE_VERSION_LABEL);
	});

	it("для отсутствия значения возвращает пустую строку", () => {
		expect(formatVersion(undefined)).toBe("");
	});
});

describe("проброс движка semver", () => {
	it("format/version отдаёт те же функции, что и logic/semver", () => {
		expect(isSemver("1.4.2")).toBe(true);
		expect(isSemver("1.4")).toBe(false);
	});
});
