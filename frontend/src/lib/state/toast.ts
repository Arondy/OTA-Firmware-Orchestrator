/**
 * Единая точка показа тостов: копии и политика живут здесь, экраны не знают
 * про svelte-sonner напрямую.
 *
 * Политика (00-CONTEXT §10.6): тост с одним ключом обновляет существующий, а не
 * складывается в стопку; сбои поллинга тостов не поднимают вовсе - для них у
 * ресурса есть инлайн-баннер.
 */
import { goto } from "$app/navigation";
import { toast as sonner } from "svelte-sonner";
import type { ApiError } from "$lib/api/errors";

function keyId(key: string | undefined): string | undefined {
	return key === undefined ? undefined : `ota:${key}`;
}

function description(hint?: string, serverMessage?: string): string | undefined {
	const parts: string[] = [];
	if (hint) parts.push(hint);
	if (serverMessage) parts.push(`Ответ сервера: ${serverMessage}`);
	return parts.length > 0 ? parts.join(" ") : undefined;
}

export const toast = {
	success(title: string, descriptionText?: string, key?: string): void {
		sonner.success(title, { id: keyId(key), description: descriptionText });
	},

	error(title: string, descriptionText?: string, key?: string): void {
		sonner.error(title, { id: keyId(key), description: descriptionText });
	},

	info(title: string, descriptionText?: string, key?: string): void {
		sonner.message(title, { id: keyId(key), description: descriptionText });
	},

	/** Сбой действия оператора: перевод, подсказка и дословный ответ сервера. */
	apiError(error: ApiError, key?: string): void {
		sonner.error(error.headline, {
			id: keyId(key),
			description: description(error.hint, error.serverMessage),
		});
	},

	/** Успех с действием-ссылкой: например, открыть созданное устройство в песочнице. */
	successAction(
		title: string,
		descriptionText: string | undefined,
		action: { label: string; href: string },
		key?: string,
	): void {
		sonner.success(title, {
			id: keyId(key),
			description: descriptionText,
			action: {
				label: action.label,
				onClick: () => {
					void goto(action.href);
				},
			},
		});
	},

	dismiss(key?: string): void {
		sonner.dismiss(keyId(key));
	},
};
