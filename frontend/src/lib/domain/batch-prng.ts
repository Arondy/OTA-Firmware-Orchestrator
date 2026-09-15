/**
 * Сеяный ГПСЧ батч-прогона песочницы.
 *
 * Процент успеха в отчётах - вход симуляции, а не измерение, поэтому исход каждой
 * попытки тянется из детерминированного генератора: прогон с тем же seed
 * воспроизводим попытка в попытку. mulberry32 выбран за простоту и отсутствие
 * зависимостей; криптостойкость не нужна и не подразумевается.
 */

/** mulberry32: состояние - одно 32-битное число, период около 2^32. */
export function mulberry32(seed: number): () => number {
	let state = seed >>> 0;
	return () => {
		state = (state + 0x6d_2b_79_f5) >>> 0;
		let t = Math.imul(state ^ (state >>> 15), 1 | state);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4_294_967_296;
	};
}

export type BatchResult = "success" | "failure";

/**
 * Исход попытки по доле успеха (0..100 целыми процентами): значение генератора
 * строго меньше доли означает успех. Граница включительна к доле: при 100%
 * ошибок не бывает, при 0% успехов нет.
 */
export function pickBatchResult(rng: () => number, successSharePercent: number): BatchResult {
	return rng() * 100 < successSharePercent ? "success" : "failure";
}
