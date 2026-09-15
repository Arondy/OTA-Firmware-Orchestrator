<script lang="ts">
	import { createDevice } from "$lib/api/endpoints";
	import { ApiError } from "$lib/api/errors";
	import type { Device } from "$lib/api/types";
	import { isSemver } from "$lib/logic/semver";
	import { shortId } from "$lib/format/id";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { toast } from "$lib/state/toast";
	import Button from "$lib/ui/Button.svelte";
	import Field from "$lib/ui/Field.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import Modal from "$lib/ui/Modal.svelte";
	import TextInput from "$lib/ui/TextInput.svelte";

	/**
	 * Регистрация устройства: тело ровно из двух ключей схемы. Подсказок под
	 * лейблами нет: ограничения оператор видит в тексте ошибки, а список
	 * моделей подсказывает datalist.
	 */
	let {
		open,
		onOpenChange,
		onCreated,
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		onCreated: (device: Device) => void;
	} = $props();

	let deviceModel = $state("");
	let currentVersion = $state("");
	let attempted = $state(false);
	let fieldErrors = $state<Partial<Record<"deviceModel" | "currentVersion", string>>>({});
	let serverError = $state<string | undefined>(undefined);

	const models = $derived(firmwareIndex.models());
	const pending = $derived(mutation.isPending("device-register"));

	const errors = $derived.by(() => {
		const next: Partial<Record<"deviceModel" | "currentVersion", string>> = {};
		const model = deviceModel.trim();
		if (model.length < 2 || model.length > 64) {
			next.deviceModel = "От 2 до 64 символов";
		}
		if (!isSemver(currentVersion.trim())) {
			next.currentVersion = "Semver, например 1.0.0";
		}
		return next;
	});

	function reset(): void {
		deviceModel = "";
		currentVersion = "";
		attempted = false;
		fieldErrors = {};
		serverError = undefined;
	}

	function errorOf(field: "deviceModel" | "currentVersion"): string | undefined {
		if (fieldErrors[field]) return fieldErrors[field];
		if (attempted) return errors[field];
		return undefined;
	}

	async function submit(): Promise<void> {
		attempted = true;
		fieldErrors = {};
		serverError = undefined;
		if (Object.keys(errors).length > 0) return;

		try {
			const created = await mutation.run(
				"device-register",
				() =>
					createDevice({
						device_model: deviceModel.trim(),
						current_version: currentVersion.trim(),
					}),
				{ notify: false },
			);
			if (!created) return;
			toast.successAction(
				"Устройство зарегистрировано",
				shortId(created.id),
				{ label: "Открыть в Песочнице", href: `/lab?device=${created.id}` },
				"device-register",
			);
			onCreated(created);
			onOpenChange(false);
			reset();
		} catch (error) {
			if (!(error instanceof ApiError)) return;
			if (error.status === 400 && error.fields) {
				const mapped: Partial<Record<"deviceModel" | "currentVersion", string>> = {};
				if (error.fields.device_model) mapped.deviceModel = error.fields.device_model;
				if (error.fields.current_version) mapped.currentVersion = error.fields.current_version;
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
	title="Регистрация устройства"
>
	<form
		class="grid gap-4"
		novalidate
		onsubmit={(event) => {
			event.preventDefault();
			void submit();
		}}
	>
		<datalist id="device-models">
			{#each models as model (model)}
				<option value={model}></option>
			{/each}
		</datalist>

		{#if serverError}
			<InlineBanner tone="danger" boxed>{serverError}</InlineBanner>
		{/if}

		<Field
			label="Модель устройства"
			for="device-register-model"
			error={errorOf("deviceModel")}
			required
		>
			<TextInput
				id="device-register-model"
				bind:value={deviceModel}
				list="device-models"
				maxlength={64}
			/>
		</Field>

		<Field
			label="Текущая версия прошивки"
			for="device-register-version"
			error={errorOf("currentVersion")}
			required
		>
			<TextInput id="device-register-version" bind:value={currentVersion} mono />
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
