/**
 * Предсказание результата чек-ина (00-CONTEXT §4).
 *
 * Сервер возвращает `update_available: false` в четырёх разных случаях и никак
 * их не различает в ответе. Экран «Песочница» обязан объяснить оператору,
 * какой именно случай сработал, поэтому проверки здесь идут строго в том же
 * порядке, что и в `UpdateService.Checkin`:
 *
 *   1. устройство выведено из эксплуатации;
 *   2. для его модели нет выполняющейся кампании;
 *   3. `current_version >= target_version` целевой прошивки;
 *   4. `bucket > target_percent` активной стадии.
 *
 * Порядок важен: сообщать «бакет не попал» устройству, которое вообще выведено
 * из эксплуатации, было бы неправдой.
 *
 * Отдельно выделен вид `unknown`: если версию не удалось разобрать или активная
 * стадия неизвестна, честный ответ - «неизвестно», а не угаданный вердикт
 * (§5.4, §10.5).
 */
import type { CampaignStatus, DeviceStatus } from "$lib/api/types";
import { bucketOf, isInBucket } from "./bucket";
import { isAtLeast, parseSemver } from "./semver";

export type CheckinDenial =
	"decommissioned" | "no_running_campaign" | "version_up_to_date" | "outside_bucket";

export type CheckinUnknown = "unparseable_version" | "no_active_stage" | "no_campaign_context";

export type CheckinKind = "available" | "denied" | "unknown";

export interface CheckinDeviceContext {
	id: string;
	device_model: string;
	status: DeviceStatus;
	current_version: string;
}

export interface CheckinStageContext {
	target_percent: number;
}

export interface CheckinCampaignContext {
	id: string;
	device_model: string;
	status: CampaignStatus;
	/** Целевая версия прошивки этой кампании. */
	target_version: string;
	/** Активная стадия; может отсутствовать, если стадия ещё не назначена. */
	active_stage?: CheckinStageContext | undefined;
}

export interface CheckinVerdict {
	kind: CheckinKind;
	/** `undefined`, когда вердикт неизвестен: интерфейс не имеет права угадать. */
	updateAvailable: boolean | undefined;
	denial?: CheckinDenial;
	unknown?: CheckinUnknown;
	/** Бакет устройства в этой кампании: считается детерминированно на клиенте. */
	bucket?: number;
	/** Короткий русский вердикт. */
	label: string;
	/** Развёрнутое пояснение для экрана «Песочница». */
	detail: string;
}

const DENIAL_COPY: Record<CheckinDenial, { label: string; detail: string }> = {
	decommissioned: {
		label: "Обновление недоступно: устройство выведено из эксплуатации",
		detail:
			"Выведенное из эксплуатации устройство не получает обновлений, даже если для его модели есть выполняющаяся кампания.",
	},
	no_running_campaign: {
		label: "Обновление недоступно: нет выполняющейся кампании",
		detail:
			"Для этой модели устройства нет кампании в статусе «выполняется». Черновики, кампании на паузе, завершённые и откаченные в чек-ине не участвуют.",
	},
	version_up_to_date: {
		label: "Обновление недоступно: версия уже не ниже целевой",
		detail: "Текущая версия устройства больше либо равна целевой версии прошивки кампании.",
	},
	outside_bucket: {
		label: "Обновление недоступно: бакет вне охвата стадии",
		detail:
			"Бакет устройства больше охвата активной стадии. Бакет не меняется между чек-инами, поэтому повторный запрос даст тот же результат.",
	},
};

const UNKNOWN_COPY: Record<CheckinUnknown, { label: string; detail: string }> = {
	unparseable_version: {
		label: "Вердикт неизвестен: версию не удалось разобрать",
		detail: "Одна из версий не соответствует формату semver, поэтому предсказание не выдаётся.",
	},
	no_active_stage: {
		label: "Вердикт неизвестен: активная стадия не задана",
		detail:
			"У кампании нет активной стадии, а значит неизвестен её охват. Без охвата нельзя проверить попадание бакета.",
	},
	no_campaign_context: {
		label: "Вердикт неизвестен: кампания не выбрана",
		detail: "Для предсказания нужны идентификатор кампании и её целевая версия прошивки.",
	},
};

function denial(reason: CheckinDenial, bucket?: number): CheckinVerdict {
	const copy = DENIAL_COPY[reason];
	return {
		kind: "denied",
		updateAvailable: false,
		denial: reason,
		bucket,
		label: copy.label,
		detail: copy.detail,
	};
}

function unknown(reason: CheckinUnknown, bucket?: number): CheckinVerdict {
	const copy = UNKNOWN_COPY[reason];
	return {
		kind: "unknown",
		updateAvailable: undefined,
		unknown: reason,
		bucket,
		label: copy.label,
		detail: copy.detail,
	};
}

/**
 * Чистая функция: не ходит в API и не читает состояние. Все входные данные
 * вызывает экран «Песочница».
 */
export function checkinVerdict(
	device: CheckinDeviceContext,
	campaign: CheckinCampaignContext | undefined,
): CheckinVerdict {
	// Проверка 1: статус устройства. Идёт первой и на сервере.
	if (device.status === "decommissioned") {
		return denial("decommissioned");
	}

	// Проверка 2: выполняющаяся кампания ровно для этой модели.
	if (!campaign) {
		return denial("no_running_campaign");
	}
	if (campaign.device_model !== device.device_model || campaign.status !== "running") {
		return denial("no_running_campaign");
	}

	const bucket = bucketOf(device.id, campaign.id);

	// Проверка 3: текущая версия уже не ниже целевой.
	const current = parseSemver(device.current_version);
	const target = parseSemver(campaign.target_version);
	if (!current || !target) {
		return unknown("unparseable_version", bucket);
	}

	if (isAtLeast(device.current_version, campaign.target_version)) {
		return denial("version_up_to_date", bucket);
	}

	// Проверка 4: бакет против охвата активной стадии.
	const stage = campaign.active_stage;
	if (!stage) {
		return unknown("no_active_stage", bucket);
	}

	if (!isInBucket(bucket, stage.target_percent)) {
		return denial("outside_bucket", bucket);
	}

	return {
		kind: "available",
		updateAvailable: true,
		bucket,
		label: "Обновление будет выдано",
		detail:
			"Устройство активно, для его модели есть выполняющаяся кампания, текущая версия ниже целевой, а бакет попадает в охват активной стадии.",
	};
}

/** Пояснение для случая «бакет попал» / «бакет не попал» в подписи к числу. */
export function bucketHint(bucket: number, targetPercent: number): string {
	return isInBucket(bucket, targetPercent)
		? `Бакет ${bucket} не больше охвата стадии ${targetPercent}%, обновление выдаётся.`
		: `Бакет ${bucket} больше охвата стадии ${targetPercent}%, обновление не выдаётся.`;
}
