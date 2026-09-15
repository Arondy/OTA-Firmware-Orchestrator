/**
 * Разбор ответа сервера на создание кампании.
 *
 * Сервер отвечает `400 { error, fields }` с ключами DTO (`firmware_version_id`,
 * `rollout_stages`) либо `400 { error }` с известной строкой. Форма обязана
 * вернуть оператора на тот шаг, где живёт offending-поле, показать ошибку у
 * ввода, а для вложенных стадий - баннером на шаге 2.
 */
import type { ApiError } from "$lib/api/errors";

export type WizardStep = 1 | 2 | 3;

export interface CreateErrorMapping {
	/** Шаг, куда возвращаем оператора. */
	step: WizardStep;
	/** Ошибки полей шага 1: ключи совпадают с именами полей формы. */
	fieldErrors: Partial<Record<"firmwareVersionId" | "rolloutStages", string>>;
	/** Текст баннера шага 2 для вложенных ошибок стадий. */
	stagesBanner?: string;
	/** Текст баннера шага создания (3) для прочих случаев. */
	banner?: string;
	/** Нужен ли баннеру действие «Обновить список прошивок». */
	refreshFirmware?: boolean;
}

/** Две строки создания, которые переводятся иначе, чем в общей таблице ошибок. */
const DUPLICATE_INDEX_MESSAGE =
	"rollout stage for this rollout campaign with such index already exists";
const FIRMWARE_NOT_FOUND_MESSAGE = "firmware version not found";

export function mapCreateError(error: ApiError): CreateErrorMapping {
	const fields = error.fields ?? {};

	if (fields.firmware_version_id) {
		return { step: 1, fieldErrors: { firmwareVersionId: fields.firmware_version_id } };
	}

	if (fields.rollout_stages) {
		return {
			step: 2,
			fieldErrors: { rolloutStages: fields.rollout_stages },
			stagesBanner: fields.rollout_stages,
		};
	}

	const serverMessage = error.serverMessage ?? "";

	if (serverMessage === DUPLICATE_INDEX_MESSAGE) {
		return {
			step: 2,
			fieldErrors: {},
			stagesBanner: "Две стадии с одинаковым порядковым номером: проверьте список стадий",
		};
	}

	if (serverMessage === FIRMWARE_NOT_FOUND_MESSAGE) {
		return {
			step: 1,
			fieldErrors: {
				firmwareVersionId: "Прошивка не найдена: возможно, она была удалена или список устарел",
			},
			refreshFirmware: true,
		};
	}

	return { step: 3, fieldErrors: {}, banner: error.headline };
}
