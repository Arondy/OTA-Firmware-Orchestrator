<script lang="ts">
	import type { Campaign } from "$lib/api/types";
	import {
		BATCH_MAX_CONCURRENCY,
		BATCH_MAX_DEVICES,
		batchRunner,
	} from "$lib/state/batch-runner.svelte";
	import { apiAvailability } from "$lib/state/api-availability.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { isSemver, parseSemver } from "$lib/logic/semver";
	import { int } from "$lib/format/number";
	import Button from "$lib/ui/Button.svelte";
	import Field from "$lib/ui/Field.svelte";
	import IndeterminateBar from "$lib/ui/IndeterminateBar.svelte";
	import NumberInput from "$lib/ui/NumberInput.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import TextInput from "$lib/ui/TextInput.svelte";

	/**
	 * Батч-прогон: флот устройств через настоящий конвейер. Процент успеха -
	 * вход симуляции, а не измерение. Пределы жёсткие: 200 устройств,
	 * 8 параллельных запросов, автоповторов нет.
	 */
	let {
		campaign,
		runningCampaignForModel,
	}: {
		campaign: Campaign | undefined;
		/** Выполняющаяся кампания для произвольной модели (из коллекции). */
		runningCampaignForModel: (model: string) => Campaign | undefined;
	} = $props();

	let count = $state(20);
	let model = $state("");
	let startVersion = $state("");
	let successShare = $state(95);
	let delayMs = $state(20);
	let concurrency = $state(4);
	let attempted = $state(false);

	function versionBelow(target: string | undefined): string {
		const parsed = parseSemver(target ?? "");
		if (!parsed) return "";
		if (parsed.major > 0) return `${parsed.major - 1}.0.0`;
		if (parsed.minor > 0) return `0.${parsed.minor - 1}.0`;
		return "0.0.0";
	}

	$effect(() => {
		if (!campaign) return;
		model = campaign.device_model;
		startVersion = versionBelow(firmwareIndex.versionLabel(campaign.firmware_version_id));
		attempted = false;
	});

	/** Кампания прогона: выбранная цель, либо running-кампания введённой модели. */
	const runCampaign = $derived.by(() => {
		if (campaign && campaign.device_model === model.trim()) return campaign;
		return runningCampaignForModel(model.trim());
	});

	const activeStage = $derived(
		runCampaign?.rollout_stages.find((stage) => stage.status === "active"),
	);

	const errors = $derived.by(() => {
		const next: Partial<Record<string, string>> = {};
		if (!Number.isInteger(count) || count < 1 || count > BATCH_MAX_DEVICES) {
			next.count = `Укажите целое число от 1 до ${BATCH_MAX_DEVICES}`;
		}
		const trimmed = model.trim();
		if (trimmed.length < 2 || trimmed.length > 64) next.model = "от 2 до 64 символов";
		if (!isSemver(startVersion.trim())) next.version = "semver, например 1.0.0";
		if (!Number.isInteger(successShare) || successShare < 0 || successShare > 100) {
			next.share = "Укажите целое число от 0 до 100";
		}
		if (!Number.isInteger(delayMs) || delayMs < 0 || delayMs > 500) {
			next.delay = "Укажите целое число от 0 до 500";
		}
		if (!Number.isInteger(concurrency) || concurrency < 1 || concurrency > BATCH_MAX_CONCURRENCY) {
			next.concurrency = `Укажите целое число от 1 до ${BATCH_MAX_CONCURRENCY}`;
		}
		return next;
	});

	const blockedReason = $derived.by(() => {
		if (batchRunner.running) return undefined;
		if (apiAvailability.down) return "API недоступен: прогон остановлен, чтобы не копить сбои";
		if (!runCampaign)
			return "для этой модели нет выполняющейся кампании: выберите кампанию или модель";
		if (!activeStage) return "у кампании нет активной стадии: обновите данные кампании";
		return undefined;
	});

	const counters = $derived(batchRunner.counters);
</script>

<Panel title="Батч-прогон">
	<div class="grid gap-4">
		<div class="flex flex-wrap items-end gap-3">
			<Field label="N устройств" for="batch-count" error={attempted ? errors.count : undefined}>
				<NumberInput id="batch-count" bind:value={count} min={1} max={BATCH_MAX_DEVICES} step={1} />
			</Field>
			<Field label="Модель" for="batch-model" error={attempted ? errors.model : undefined}>
				<TextInput id="batch-model" bind:value={model} maxlength={64} class="w-52" />
			</Field>
			<Field
				label="Начальная версия"
				for="batch-version"
				error={attempted ? errors.version : undefined}
			>
				<TextInput id="batch-version" bind:value={startVersion} mono class="w-32" />
			</Field>
			<Field
				label="Процент успеха в отчётах"
				for="batch-share"
				error={attempted ? errors.share : undefined}
			>
				<NumberInput id="batch-share" bind:value={successShare} min={0} max={100} step={1}>
					{#snippet unit()}%{/snippet}
				</NumberInput>
			</Field>
			<Field
				label="Задержка между запросами"
				for="batch-delay"
				error={attempted ? errors.delay : undefined}
			>
				<NumberInput id="batch-delay" bind:value={delayMs} min={0} max={500} step={10}>
					{#snippet unit()}мс{/snippet}
				</NumberInput>
			</Field>
			<Field
				label="Параллельно"
				for="batch-concurrency"
				error={attempted ? errors.concurrency : undefined}
			>
				<NumberInput
					id="batch-concurrency"
					bind:value={concurrency}
					min={1}
					max={BATCH_MAX_CONCURRENCY}
					step={1}
				/>
			</Field>
		</div>

		{#if blockedReason}
			<p class="text-dense text-fg-muted">{blockedReason}</p>
		{/if}

		<div class="flex flex-wrap items-center gap-2">
			<Button
				variant="primary"
				disabled={blockedReason !== undefined}
				onclick={() => {
					attempted = true;
					if (Object.keys(errors).length > 0 || !runCampaign || !activeStage) return;
					void batchRunner.start({
						count,
						model: model.trim(),
						startVersion: startVersion.trim(),
						successSharePercent: successShare,
						delayMs,
						concurrency,
						campaignId: runCampaign.id,
						stageId: activeStage.id,
						seed: Date.now() % 2 ** 31,
					});
				}}
			>
				Прогнать
			</Button>
			{#if batchRunner.running}
				<Button variant="danger" onclick={() => batchRunner.stop()}>Остановить</Button>
			{/if}
		</div>

		{#if batchRunner.phase !== "idle"}
			<div class="grid gap-2 border-t border-border-subtle pt-3">
				{#if batchRunner.running}
					<IndeterminateBar label="Батч-прогон выполняется" />
				{/if}
				<p class="text-table text-fg-primary tabular-nums">
					{int(batchRunner.processed)} / {int(batchRunner.total)} устройств
				</p>
				<p class="flex flex-wrap gap-x-4 gap-y-1 text-dense text-fg-secondary tabular-nums">
					<span>зарегистрировано {int(counters.registered)}</span>
					<span>обновление выдано {int(counters.updated)}</span>
					<span>вне выборки {int(counters.outOfSample)}</span>
					<span>успех {int(counters.success)}</span>
					<span>ошибка {int(counters.failure)}</span>
					<span>таймаут {int(counters.timeout)}</span>
					<span>сбой запроса {int(counters.requestError)}</span>
				</p>
				{#if batchRunner.summary && runCampaign}
					<a
						href="/campaigns/{runCampaign.id}"
						class="text-table underline underline-offset-4 hover:text-accent-text"
					>
						Открыть кампанию
					</a>
				{/if}
			</div>
		{/if}
	</div>
</Panel>
