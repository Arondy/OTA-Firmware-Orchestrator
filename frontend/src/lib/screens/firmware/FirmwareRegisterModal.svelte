<script lang="ts">
	import { createFirmwareVersion } from "$lib/api/endpoints";
	import { ApiError } from "$lib/api/errors";
	import type { FirmwareVersion } from "$lib/api/types";
	import {
		buildFirmwareBody,
		EMPTY_FIRMWARE_DRAFT,
		validateFirmwareDraft,
		type FirmwareDraft,
		type FirmwareField,
	} from "$lib/domain/firmware-draft";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { toast } from "$lib/state/toast";
	import Button from "$lib/ui/Button.svelte";
	import Field from "$lib/ui/Field.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import Modal from "$lib/ui/Modal.svelte";
	import TextInput from "$lib/ui/TextInput.svelte";

	/**
	 * Регистрация прошивки: тело запроса ровно из четырёх ключей схемы,
	 * семвер проверяется тем же регулярным выражением, что примет сервер,
	 * сумма нормируется к нижнему регистру. На 409 даём ссылку на уже
	 * существующую запись реестра, на 400 с полями - ошибки у вводов.
	 */
	let {
		open,
		onOpenChange,
		onCreated,
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		/** Куда вернуть созданную запись: мастер подставит её в форму. */
		onCreated?: (version: FirmwareVersion) => void;
	} = $props();

	let draft = $state<FirmwareDraft>({ ...EMPTY_FIRMWARE_DRAFT });
	let attempted = $state(false);
	let conflict = $state<FirmwareVersion | undefined>(undefined);
	let serverError = $state<string | undefined>(undefined);
	let fieldErrors = $state<Partial<Record<FirmwareField, string>>>({});

	const errors = $derived(validateFirmwareDraft(draft));
	const checksumLength = $derived(draft.fwChecksum.replace(/\s/g, "").length);
	const pending = $derived(mutation.isPending("firmware-register"));
	const models = $derived(firmwareIndex.models());

	function reset(): void {
		draft = { ...EMPTY_FIRMWARE_DRAFT };
		attempted = false;
		conflict = undefined;
		serverError = undefined;
		fieldErrors = {};
	}

	function errorOf(field: FirmwareField): string | undefined {
		if (fieldErrors[field]) return fieldErrors[field];
		if (attempted) return errors[field];
		return undefined;
	}

	async function submit(): Promise<void> {
		attempted = true;
		conflict = undefined;
		serverError = undefined;
		fieldErrors = {};
		if (Object.keys(errors).length > 0) return;

		try {
			const created = await mutation.run(
				"firmware-register",
				() => createFirmwareVersion(buildFirmwareBody(draft)),
				{ notify: false },
			);
			if (!created) return;
			await firmwareIndex.refresh();
			toast.success(
				"Прошивка зарегистрирована",
				`${created.device_model} ${created.fw_version}`,
				"firmware-register",
			);
			onCreated?.(created);
			onOpenChange(false);
			reset();
		} catch (error) {
			if (!(error instanceof ApiError)) return;
			if (error.status === 409) {
				const body = buildFirmwareBody(draft);
				conflict = firmwareIndex.versions.find(
					(version) =>
						version.device_model === body.device_model && version.fw_version === body.fw_version,
				);
				return;
			}
			if (error.status === 400 && error.fields) {
				const mapped: Partial<Record<FirmwareField, string>> = {};
				if (error.fields.device_model) mapped.deviceModel = error.fields.device_model;
				if (error.fields.fw_version) mapped.fwVersion = error.fields.fw_version;
				if (error.fields.fw_checksum) mapped.fwChecksum = error.fields.fw_checksum;
				if (error.fields.binary_url) mapped.binaryUrl = error.fields.binary_url;
				fieldErrors = mapped;
				return;
			}
			serverError = error.headline;
		}
	}
</script>

<Modal
	{open}
	onOpenChange={(next) => {
		if (!next && pending) return;
		onOpenChange(next);
		if (!next) reset();
	}}
	title="Регистрация прошивки"
>
	<form
		class="grid gap-4"
		novalidate
		onsubmit={(event) => {
			event.preventDefault();
			void submit();
		}}
	>
		<datalist id="firmware-models">
			{#each models as model (model)}
				<option value={model}></option>
			{/each}
		</datalist>

		{#if conflict}
			<InlineBanner tone="danger" boxed>
				<span class="flex flex-wrap items-center gap-x-2 gap-y-1">
					Такая версия для этой модели уже зарегистрирована.
					<a
						href="/firmware?model={encodeURIComponent(conflict.device_model)}"
						class="underline underline-offset-4 hover:text-accent-text"
					>
						Открыть в реестре
					</a>
				</span>
			</InlineBanner>
		{/if}

		{#if serverError}
			<InlineBanner tone="danger" boxed>{serverError}</InlineBanner>
		{/if}

		<Field label="Модель устройства" for="firmware-model" error={errorOf("deviceModel")} required>
			<TextInput
				id="firmware-model"
				bind:value={draft.deviceModel}
				list="firmware-models"
				maxlength={64}
			/>
		</Field>

		<Field label="Версия прошивки" for="firmware-version" error={errorOf("fwVersion")} required>
			<TextInput id="firmware-version" bind:value={draft.fwVersion} mono />
		</Field>

		<Field label="Контрольная сумма" for="firmware-checksum" error={errorOf("fwChecksum")} required>
			<TextInput id="firmware-checksum" bind:value={draft.fwChecksum} mono maxlength={64} />
			<span class="text-dense text-fg-muted tabular-nums">{checksumLength} / 64</span>
		</Field>

		<Field label="Ссылка на бинарник" for="firmware-url" error={errorOf("binaryUrl")} required>
			<TextInput id="firmware-url" bind:value={draft.binaryUrl} type="url" mono />
		</Field>

		<div class="flex justify-end gap-2">
			<Button
				type="button"
				variant="secondary"
				disabled={pending}
				onclick={() => onOpenChange(false)}
			>
				Отмена
			</Button>
			<Button type="submit" variant="primary" loading={pending}>Зарегистрировать</Button>
		</div>
	</form>
</Modal>
