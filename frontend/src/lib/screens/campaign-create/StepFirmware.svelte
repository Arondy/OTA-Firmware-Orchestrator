<script lang="ts">
	import BinaryIcon from "phosphor-svelte/lib/BinaryIcon";
	import { createResource } from "$lib/state/resource.svelte";
	import { listFirmwareVersions } from "$lib/api/endpoints";
	import type { FirmwareVersion } from "$lib/api/types";
	import { campaignDraft } from "$lib/state/campaign-draft.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { compareSemver } from "$lib/logic/semver";
	import { relativeTime } from "$lib/format/datetime";
	import { clock } from "$lib/state/clock.svelte";
	import Button from "$lib/ui/Button.svelte";
	import ErrorState from "$lib/ui/ErrorState.svelte";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import ExternalLink from "$lib/ui/ExternalLink.svelte";
	import Field from "$lib/ui/Field.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import SearchSelect from "$lib/ui/SearchSelect.svelte";
	import Select from "$lib/ui/Select.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";

	/**
	 * Шаг 1: модель и версия прошивки. Список моделей собран из реестра
	 * прошивок (отдельного эндпоинта с моделями в контракте нет), версии
	 * догружаются по модели не более чем тремя страницами и отсортированы
	 * по semver от новых к старым.
	 */
	let {
		models,
		runningModelCampaignId,
		fieldError,
		refreshFirmware,
		onRegister,
		onRefreshFirmware,
	}: {
		models: string[];
		/** id выполняющейся кампании выбранной модели, если она известна. */
		runningModelCampaignId: string | undefined;
		/** Ошибка поля с сервера (400 fields.firmware_version_id или перевод). */
		fieldError: string | undefined;
		/** Сервер не нашёл прошивку: списку нужно обновление. */
		refreshFirmware: boolean;
		onRegister: () => void;
		onRefreshFirmware: () => void;
	} = $props();

	/** Версии выбранной модели: не больше трёх страниц по 100 (§5.2). */
	const versionsResource = createResource<FirmwareVersion[]>(
		"campaign-create-model-versions",
		async (signal) => {
			const model = campaignDraft.model;
			if (!model) return [];
			const rows: FirmwareVersion[] = [];
			for (let page = 1; page <= 3; page += 1) {
				const response = await listFirmwareVersions(
					{ device_model: model, page, limit: 100 },
					{ signal },
				);
				rows.push(...response.firmware_versions);
				if (response.firmware_versions.length < 100) break;
			}
			return rows;
		},
	);

	let lastModel = $state<string | null>(null);
	$effect(() => {
		// Перечитываем версии при смене модели; сброс выбранной версии делает
		// сам черновик (сеттер модели), а не этот эффект.
		const model = campaignDraft.model;
		if (model === lastModel) return;
		lastModel = model;
		void versionsResource.refresh();
	});

	const versionItems = $derived.by(() => {
		const rows = [...(versionsResource.data ?? [])];
		// Только что зарегистрированная версия могла не попасть в скачанный
		// список: добавляем выбранную запись из индекса, чтобы Select её видел.
		const chosen = firmwareIndex.get(campaignDraft.firmwareVersionId);
		if (chosen && !rows.some((row) => row.id === chosen.id)) rows.push(chosen);
		return rows
			.sort((a, b) => -compareSemver(a.fw_version, b.fw_version))
			.map((version) => ({
				value: version.id,
				label: `${version.fw_version}, ${relativeTime(version.created_at, clock.now)}`,
			}));
	});

	const selected = $derived(firmwareIndex.get(campaignDraft.firmwareVersionId));
	const modelItems = $derived(models.map((model) => ({ value: model, label: model })));
</script>

{#if firmwareIndex.error && firmwareIndex.versions.length === 0}
	<ErrorState error={firmwareIndex.error} onretry={() => void firmwareIndex.refresh()} />
{:else if firmwareIndex.versions.length === 0 && !firmwareIndex.pending}
	<EmptyState
		icon={BinaryIcon}
		title="Прошивок пока нет"
		text="Зарегистрируйте первую версию: модель, semver, sha256 и ссылка на бинарник."
	>
		{#snippet action()}
			<Button variant="primary" onclick={onRegister}>Зарегистрировать прошивку</Button>
		{/snippet}
	</EmptyState>
{:else}
	<div class="grid gap-4">
		<p class="text-dense text-fg-muted">черновик не сохраняется между перезагрузками страницы</p>

		{#if campaignDraft.firmwareNotice}
			<InlineBanner tone="warning" boxed>{campaignDraft.firmwareNotice}</InlineBanner>
		{/if}

		{#if runningModelCampaignId}
			<InlineBanner tone="warning" boxed>
				<span class="flex flex-wrap items-center gap-x-2 gap-y-1">
					Для этой модели уже выполняется кампания.
					<a
						href="/campaigns/{runningModelCampaignId}"
						class="underline underline-offset-4 hover:text-accent-text"
					>
						Открыть её
					</a>
				</span>
			</InlineBanner>
		{/if}

		<Field label="Модель" required>
			<SearchSelect
				id="create-model"
				bind:value={campaignDraft.model}
				items={modelItems}
				ariaLabel="Модель устройства"
				placeholder="Выберите модель устройства"
			>
				{#snippet footer()}
					<Button variant="ghost" size="sm" onclick={onRegister}>Зарегистрировать прошивку</Button>
				{/snippet}
			</SearchSelect>
		</Field>

		<Field
			label="Версия прошивки"
			hint="Для одной модели одновременно может выполняться только одна кампания."
			error={fieldError}
			required
		>
			{#if versionsResource.pending && campaignDraft.model}
				<Skeleton class="h-9 w-full" />
			{:else}
				<Select
					id="create-version"
					bind:value={campaignDraft.firmwareVersionId}
					items={versionItems}
					ariaLabel="Версия прошивки"
					placeholder={campaignDraft.model ? "Выберите версию" : "Сначала выберите модель"}
					disabled={!campaignDraft.model}
				/>
			{/if}
		</Field>

		{#if refreshFirmware}
			<Button variant="secondary" size="sm" onclick={onRefreshFirmware}>
				Обновить список прошивок
			</Button>
		{/if}

		{#if selected}
			<div class="grid gap-1.5 border-t border-border-subtle pt-3">
				<div class="flex flex-wrap items-center gap-x-4 gap-y-1.5">
					<span class="flex items-center gap-1.5">
						<span class="text-micro text-fg-muted">Контрольная сумма</span>
						<CopyButton
							text={selected.fw_checksum}
							copyKey="create-firmware-checksum"
							display={`${selected.fw_checksum.slice(0, 8)}…${selected.fw_checksum.slice(-8)}`}
						/>
					</span>
					<span class="flex items-center gap-1.5">
						<span class="text-micro text-fg-muted">Бинарник</span>
						<ExternalLink href={selected.binary_url} class="max-w-72 truncate">
							{selected.binary_url}
						</ExternalLink>
					</span>
				</div>
				<span class="flex items-center gap-1.5">
					<span class="text-micro text-fg-muted">Идентификатор</span>
					<MonoId id={selected.id} copyKey="create-firmware-id" />
				</span>
			</div>
		{/if}
	</div>
{/if}
