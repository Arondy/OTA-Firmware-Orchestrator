import { describe, expect, it } from "vitest";
import {
	compareSemver,
	isAtLeast,
	isSemver,
	parseSemver,
	semverSortKey,
	tryCompareSemver,
} from "./semver";

describe("isSemver", () => {
	it("принимает формы, разрешённые регулярным выражением из openapi.yaml", () => {
		for (const value of [
			"0.0.0",
			"1.0.0",
			"1.2.3-rc.1",
			"1.2.3+build.5",
			"1.4.2",
			"10.20.30",
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-0.3.7",
			"1.0.0-x.7.z.92",
			"1.0.0+20130313144700",
			"1.0.0-beta+exp.sha.5114f85",
		]) {
			expect(isSemver(value), value).toBe(true);
		}
	});

	it("отклоняет всё, что сервер не принял бы", () => {
		for (const value of [
			"",
			"1",
			"1.0",
			"1.2",
			"v1.4.2",
			"01.2.3",
			"1.02.3",
			"1.2.3-",
			"1.2.3+",
			"1.2.3-01",
			"latest",
			"1.2.3.4",
		]) {
			expect(isSemver(value), value).toBe(false);
		}
	});
});

describe("parseSemver", () => {
	it("раскладывает версию на части", () => {
		expect(parseSemver("1.4.2")).toEqual({
			major: 1,
			minor: 4,
			patch: 2,
			prerelease: [],
			build: [],
			raw: "1.4.2",
		});
	});

	it("отделяет pre-release и метаданные сборки", () => {
		const parsed = parseSemver("1.0.0-rc.1+build.5114f85");
		expect(parsed?.prerelease).toEqual(["rc", "1"]);
		expect(parsed?.build).toEqual(["build", "5114f85"]);
	});

	it("возвращает undefined для мусора", () => {
		expect(parseSemver("не версия")).toBeUndefined();
	});
});

describe("compareSemver", () => {
	it("сравнивает основные номера", () => {
		expect(compareSemver("2.0.0", "1.9.9")).toBe(1);
		expect(compareSemver("1.0.0", "1.0.1")).toBe(-1);
		expect(compareSemver("1.2.3", "1.2.3")).toBe(0);
	});

	it("версия без pre-release старше версии с pre-release", () => {
		expect(compareSemver("1.0.0", "1.0.0-alpha")).toBe(1);
		expect(compareSemver("1.0.0-alpha", "1.0.0")).toBe(-1);
	});

	it("числовой идентификатор pre-release младше буквенно-цифрового", () => {
		expect(compareSemver("1.0.0-1", "1.0.0-alpha")).toBe(-1);
	});

	it("числовые идентификаторы сравниваются как числа, а не лексически", () => {
		expect(compareSemver("1.0.0-2", "1.0.0-10")).toBe(-1);
	});

	it("меньшее число идентификаторов pre-release младше при равном префиксе", () => {
		expect(compareSemver("1.0.0-alpha", "1.0.0-alpha.1")).toBe(-1);
	});

	it("метаданные сборки не влияют на порядок", () => {
		expect(compareSemver("1.0.0+build.1", "1.0.0+build.999")).toBe(0);
		expect(compareSemver("1.0.0+a", "1.0.0")).toBe(0);
	});

	it("классический пример порядка из SemVer 2.0.0", () => {
		const ordered = [
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-alpha.beta",
			"1.0.0-beta",
			"1.0.0-beta.2",
			"1.0.0-beta.11",
			"1.0.0-rc.1",
			"1.0.0",
		];
		for (let index = 1; index < ordered.length; index += 1) {
			expect(compareSemver(ordered[index - 1], ordered[index])).toBe(-1);
		}
	});

	it("бросает RangeError на неразобранной версии", () => {
		expect(() => compareSemver("1.0.0", "latest")).toThrow(RangeError);
	});
});

describe("tryCompareSemver", () => {
	it("возвращает undefined вместо исключения", () => {
		expect(tryCompareSemver("1.0.0", "latest")).toBeUndefined();
		expect(tryCompareSemver("1.0.0", "1.0.1")).toBe(-1);
	});
});

describe("isAtLeast", () => {
	it("повторяет серверное условие GreaterThanEqual", () => {
		expect(isAtLeast("1.4.2", "1.4.2")).toBe(true);
		expect(isAtLeast("1.5.0", "1.4.2")).toBe(true);
		expect(isAtLeast("1.4.1", "1.4.2")).toBe(false);
	});

	it("не угадывает, когда версию не разобрать", () => {
		expect(isAtLeast("1.4.2", "latest")).toBeUndefined();
	});
});

describe("semverSortKey", () => {
	it("числовые части дополняются: 1.10.0 старше 1.9.0 в лексической сортировке", () => {
		expect(semverSortKey("1.10.0") > semverSortKey("1.9.0")).toBe(true);
	});

	it("pre-release младше релиза той же версии", () => {
		expect(semverSortKey("1.0.0-rc.1") < semverSortKey("1.0.0")).toBe(true);
	});

	it("неразобранные строки получают пустой ключ", () => {
		expect(semverSortKey("latest")).toBe("");
	});
});
