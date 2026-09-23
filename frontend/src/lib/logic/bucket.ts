/**
 * Детерминированный бакет устройства в раскатке.
 *
 * Реализация повторяет серверную один в один. Оригинал
 * (`ota-orchestrator/internal/core/service/update/update.go`):
 *
 *     func (s *UpdateService) calculateBucket(deviceID, campaignID uuid.UUID) uint32 {
 *         hash := fnv.New32a()
 *         hash.Reset()
 *         hash.Write(deviceID[:])
 *         hash.Write(campaignID[:])
 *         return (hash.Sum32() % 100) + 1
 *     }
 *
 * `uuid.UUID` в Go - это `[16]byte` в порядке канонической записи, то есть
 * байты идут ровно в том порядке, в котором шестнадцатеричные цифры стоят в
 * строке UUID без дефисов. Поэтому клиент может предсказать бакет без запроса
 * к серверу, и экран «Песочница» показывает честное значение, а не приближение.
 *
 * `Math.imul` обязателен: обычное умножение в JS теряет старшие биты, а `>>> 0`
 * приводит результат к беззнаковому 32-битному, как `uint32` в Go.
 */

const FNV_OFFSET_BASIS = 0x811c9dc5;
const FNV_PRIME = 0x01000193;

const HEX_PAIR = /^[0-9a-fA-F]{2}$/;

export const BUCKET_MIN = 1;
export const BUCKET_MAX = 100;

/**
 * Преобразует канонический UUID в 16 сырых байт.
 * Принимает и запись с дефисами, и 32 шестнадцатеричные цифры без них.
 */
export function uuidToBytes(uuid: string): Uint8Array {
	const hex = uuid.replace(/-/g, "");

	if (hex.length !== 32) {
		throw new Error(`ожидается UUID из 32 шестнадцатеричных цифр, получено: ${uuid}`);
	}

	const bytes = new Uint8Array(16);
	for (let index = 0; index < 16; index += 1) {
		const pair = hex.slice(index * 2, index * 2 + 2);
		if (!HEX_PAIR.test(pair)) {
			throw new Error(`не шестнадцатеричный UUID: ${uuid}`);
		}
		bytes[index] = Number.parseInt(pair, 16);
	}

	return bytes;
}

/**
 * Золотые значения. Посчитаны дважды - этой функцией и независимой реализацией
 * FNV-1a на Python поверх тех же 32 байт - и совпали побайтово; тест
 * `bucket.test.ts` их закрепляет. Если серверный алгоритм изменится, тест упадёт.
 *
 *   device 3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c
 *   campaign 9b7f2a10-4c3d-4e5f-8a6b-1c2d3e4f5a6b
 *     байты   3f6d2d8e4b1c4a5f9a7b8e7d1c2a3b4c9b7f2a104c3d4e5f8a6b1c2d3e4f5a6b
 *     fnv1a32 2617095165 (0x9bfdb3fd)  bucket 66
 *
 *   device 01990f2a-7c6b-7e11-9a3f-2b8c4d5e6f70
 *   campaign 01990f2a-9d1e-7a44-8b02-4c6f1a2b3c4d
 *     fnv1a32 3338382328 (0xc6fbabf8)  bucket 29
 *
 *   оба UUID нулевые: fnv1a32 187360325 (0x0b2ae445), bucket 26
 *   пустой вход:      fnv1a32 2166136261 (0x811c9dc5) - исходное смещение FNV-1a
 */

/** FNV-1a, 32 бита. Возвращает беззнаковое значение, как `hash.Sum32()` в Go. */
export function fnv1a32(bytes: Uint8Array): number {
	let hash = FNV_OFFSET_BASIS;

	for (let index = 0; index < bytes.length; index += 1) {
		hash ^= bytes[index];
		hash = Math.imul(hash, FNV_PRIME) >>> 0;
	}

	return hash >>> 0;
}

/**
 * Бакет устройства в конкретной кампании: число от 1 до 100.
 * Обновление выдаётся, когда `bucket <= target_percent` активной стадии.
 */
export function bucketOf(deviceId: string, campaignId: string): number {
	const bytes = new Uint8Array(32);
	bytes.set(uuidToBytes(deviceId), 0);
	bytes.set(uuidToBytes(campaignId), 16);

	return (fnv1a32(bytes) % 100) + 1;
}

/** Попадает ли бакет в охват стадии. */
export function isInBucket(bucket: number, targetPercent: number): boolean {
	return bucket <= targetPercent;
}
