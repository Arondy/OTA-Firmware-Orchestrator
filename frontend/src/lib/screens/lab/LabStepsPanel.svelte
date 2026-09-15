<script lang="ts">
	import type { Campaign, CheckinResult, Device, UpdateAttempt } from "$lib/api/types";
	import { checkinDevice, reportUpdateAttempt } from "$lib/api/endpoints";
	import { ApiError } from "$lib/api/errors";
	import { ATTEMPT_RESULT } from "$lib/domain/status";
	import { checkinVerdict, type CheckinVerdict } from "$lib/logic/checkin-verdict";
	import { middleTruncate, shortId } from "$lib/format/id";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { batchRunner } from "$lib/state/batch-runner.svelte";
	import Button from "$lib/ui/Button.svelte";
	import ExternalLink from "$lib/ui/ExternalLink.svelte";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import JsonView from "$lib/ui/JsonView.svelte";
	import KeyValue from "$lib/ui/KeyValue.svelte";
	import KeyValueGrid from "$lib/ui/KeyValueGrid.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import SegmentedControl from "$lib/ui/SegmentedControl.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Textarea from "$lib/ui/Textarea.svelte";
	import Field from "$lib/ui/Field.svelte";

	/**
	 * Три настоящих запроса: регистрация, чек-ин, отчёт. Шаги открываются по
	 * порядку, но перезапускаются в любой момент. Причина `update_available:
	 * false` вычисляется на клиенте тем же порядком проверок, что и на
	 * сервере, и всегда помечена как вычисленная (§4).
	 */
	let {
		device,
		campaign,
		registerResult,
		onRegister,
		onCheckin,
	}: {
		device: Device | undefined;
		campaign: Campaign | undefined;
		registerResult: Device | undefined;
		onRegister: () => Promise<Device | undefined>;
		onCheckin?: () => void;
	} = $props();

	let checkin = $state<CheckinResult | undefined>(undefined);
	let verdict = $state<CheckinVerdict | undefined>(undefined);
	let reportResultKind = $state<"success" | "failure" | "timeout">("success");
	let reportMessage = $state("");
	let attempt = $state<UpdateAttempt | undefined>(undefined);
	let reportError = $state<ApiError | undefined>(undefined);

	const activeStage = $derived(campaign?.rollout_stages.find((stage) => stage.status === "active"));
	const targetVersion = $derived(firmwareIndex.versionLabel(campaign?.firmware_version_id));
	const stageId = $derived(checkin?.stage_id);
	const isDraft = $derived(campaign?.status === "draft");
	const isPaused = $derived(campaign?.status === "paused");

	const pendingRegister = $derived(mutation.isPending("lab-register"));
	const pendingCheckin = $derived(mutation.isPending("lab-checkin"));
	const pendingReport = $derived(mutation.isPending("lab-report"));

	async function runCheckin(): Promise<void> {
		if (!device) return;
		reportError = undefined;
		try {
			const result = await mutation.run("lab-checkin", () =>
				checkinDevice(device.id, { current_version: device.current_version }),
			);
			if (!result) return;
			batchRunner.noteManual(
				"checkin",
				device.id,
				200,
				`update_available=${String(result.update_available)}`,
				result.update_available ? "success" : "neutral",
			);
			checkin = result;
			attempt = undefined;
			onCheckin?.();
			verdict = checkinVerdict(
				{
					id: device.id,
					device_model: device.device_model,
					status: device.status,
					current_version: device.current_version,
				},
				campaign
					? {
							id: campaign.id,
							device_model: campaign.device_model,
							status: campaign.status,
							target_version: targetVersion ?? "",
							active_stage: activeStage
								? { target_percent: activeStage.target_percent }
								: undefined,
						}
					: undefined,
			);
		} catch (error) {
			checkin = undefined;
			verdict = undefined;
			const status = error instanceof ApiError ? error.status : undefined;
			const message =
				error instanceof ApiError ? (error.serverMessage ?? error.headline) : String(error);
			batchRunner.noteManual("checkin", device.id, status, `сбой: ${message}`, "danger");
		}
	}

	async function runReport(): Promise<void> {
		if (!device || !campaign || !stageId || !activeStage) return;
		reportError = undefined;
		try {
			const created = await mutation.run("lab-report", () =>
				reportUpdateAttempt(device.id, {
					campaign_id: campaign.id,
					stage_id: stageId,
					result: reportResultKind,
					...(reportResultKind !== "success" && reportMessage.trim()
						? { error_message: reportMessage.trim() }
						: {}),
				}),
			);
			if (!created) return;
			batchRunner.noteManual(
				"report",
				device.id,
				200,
				reportResultKind,
				reportResultKind === "success" ? "success" : "danger",
			);
			attempt = created;
		} catch (error) {
			attempt = undefined;
			if (error instanceof ApiError) {
				reportError = error;
				batchRunner.noteManual(
					"report",
					device.id,
					error.status,
					error.serverMessage ?? error.headline,
					"danger",
				);
			}
		}
	}
</script>

<Panel title="Шаги">
	<ol class="grid gap-4">
		<li class="grid gap-2 rounded-panel border border-border-subtle bg-bg-surface p-3">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<span class="text-dense font-medium text-fg-primary">Шаг 1. Регистрация</span>
				<Button
					variant="secondary"
					size="sm"
					loading={pendingRegister}
					onclick={() => void onRegister()}
				>
					Зарегистрировать устройство
				</Button>
			</div>
			{#if registerResult}
				<!-- Одна колонка: в трёх идентификатор не влезал и обрезался
				     до середины первого символа. -->
				<KeyValueGrid>
					<KeyValue label="Идентификатор"
						><MonoId id={registerResult.id} copyKey="lab-step1-id" /></KeyValue
					>
					<KeyValue label="Модель">{registerResult.device_model}</KeyValue>
					<KeyValue label="Версия" mono>{registerResult.current_version}</KeyValue>
				</KeyValueGrid>
				<JsonView value={registerResult} label="POST /api/v1/devices" />
			{/if}
		</li>

		<li class="grid gap-2 rounded-panel border border-border-subtle bg-bg-surface p-3">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<span class="text-dense font-medium text-fg-primary">Шаг 2. Чек-ин</span>
				<Button
					variant="secondary"
					size="sm"
					disabled={!device}
					loading={pendingCheckin}
					onclick={() => void runCheckin()}
				>
					Выполнить чек-ин
				</Button>
			</div>
			{#if !device}
				<p class="text-dense text-fg-muted">сначала выберите или зарегистрируйте устройство</p>
			{:else if checkin && verdict}
				{#if checkin.update_available}
					<div class="grid gap-2 rounded-chip border border-state-success/35 bg-bg-inset p-2">
						<span class="flex items-center gap-2 text-table font-medium text-fg-primary">
							<span class="size-1.5 rounded-full bg-state-success" aria-hidden="true"></span>
							Обновление доступно
						</span>
						<KeyValueGrid>
							<KeyValue label="Версия" mono>{targetVersion ?? "неизвестна"}</KeyValue>
							<KeyValue label="Стадия">
								{#if checkin.stage_id}
									<MonoId id={checkin.stage_id} copyKey="lab-step2-stage" />
								{:else}
									нет
								{/if}
							</KeyValue>
							<KeyValue label="Контрольная сумма">
								{#if checkin.fw_checksum}
									<CopyButton
										text={checkin.fw_checksum}
										copyKey="lab-step2-checksum"
										display={`${checkin.fw_checksum.slice(0, 8)}…${checkin.fw_checksum.slice(-8)}`}
									/>
								{:else}
									нет
								{/if}
							</KeyValue>
							<KeyValue label="Бинарник">
								{#if checkin.binary_url}
									<Tooltip label={checkin.binary_url}>
										<span class="inline-flex min-w-0 max-w-full">
											<ExternalLink href={checkin.binary_url}>
												<span class="max-w-[32ch] truncate">
													{middleTruncate(checkin.binary_url, 40)}
												</span>
											</ExternalLink>
										</span>
									</Tooltip>
								{:else}
									нет
								{/if}
							</KeyValue>
						</KeyValueGrid>
						<JsonView value={checkin} label="POST /api/v1/devices/{device.id}/checkin" />
					</div>
				{:else}
					<div class="grid gap-2 rounded-chip border border-border-subtle bg-bg-inset p-2">
						<span class="flex items-center gap-2 text-table font-medium text-fg-primary">
							<span class="size-1.5 rounded-full bg-state-neutral" aria-hidden="true"></span>
							Обновление не выдано
						</span>
						<p class="text-table text-fg-secondary">{verdict.label}</p>
						<p class="text-dense text-fg-secondary">{verdict.detail}</p>
						{#if verdict.kind === "unknown"}
							<p class="text-dense text-fg-muted">причину определить не удалось</p>
						{/if}
						<JsonView value={checkin} label="POST /api/v1/devices/{device.id}/checkin" />
					</div>
				{/if}
			{/if}
		</li>

		<li class="grid gap-2 rounded-panel border border-border-subtle bg-bg-surface p-3">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<span class="text-dense font-medium text-fg-primary">Шаг 3. Отчёт</span>
				<Button
					variant="secondary"
					size="sm"
					disabled={!device || !stageId || isDraft}
					loading={pendingReport}
					onclick={() => void runReport()}
				>
					Отправить отчёт
				</Button>
			</div>

			{#if isDraft}
				<p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-dense text-fg-muted">
					Отчёт доступен после старта кампании.
					<a
						href="/campaigns/{campaign?.id}"
						class="underline underline-offset-4 hover:text-accent-text"
					>
						Запустить
					</a>
				</p>
			{:else if !stageId}
				<p class="text-dense text-fg-muted">
					нужен чек-ин с выданным обновлением: из него берутся кампания и стадия
				</p>
			{:else}
				{#if isPaused}
					<InlineBanner tone="info" boxed>
						Кампания на паузе: отчёт попадёт в счётчики стадии, но стадии не продвигаются до
						возобновления
					</InlineBanner>
				{/if}

				<div class="grid gap-3 sm:grid-cols-2">
					<div class="grid content-start gap-1.5">
						<span class="text-dense font-medium text-fg-secondary">Результат</span>
						<SegmentedControl
							name="lab-report-result"
							value={reportResultKind}
							ariaLabel="Результат попытки обновления"
							items={[
								{ value: "success", label: "Успех" },
								{ value: "failure", label: "Ошибка" },
								{ value: "timeout", label: "Таймаут" },
							]}
							onchange={(value) => {
								reportResultKind = value as typeof reportResultKind;
							}}
						/>
					</div>
					<!-- Сообщение об ошибке нужно только ошибке и таймауту:
					     при «успехе» поле не показывается вовсе. -->
					{#if reportResultKind !== "success"}
						<Field label="Сообщение об ошибке" for="lab-report-message" hint="необязательно">
							<Textarea
								id="lab-report-message"
								bind:value={reportMessage}
								rows={2}
								maxlength={1024}
							/>
							<span class="text-dense text-fg-muted tabular-nums">
								{reportMessage.length} / 1024
							</span>
						</Field>
					{/if}
				</div>

				<KeyValueGrid class="sm:grid-cols-2">
					<KeyValue label="Кампания"
						><MonoId id={campaign?.id ?? ""} copyKey="lab-step3-campaign" /></KeyValue
					>
					<KeyValue label="Стадия"><MonoId id={stageId} copyKey="lab-step3-stage" /></KeyValue>
				</KeyValueGrid>

				{#if reportError}
					<InlineBanner tone="danger" boxed>
						<span class="grid gap-1">
							<span class="font-medium text-fg-primary">{reportError.headline}</span>
							{#if reportError.hint}<span>{reportError.hint}</span>{/if}
						</span>
					</InlineBanner>
				{/if}

				{#if attempt}
					<KeyValueGrid class="sm:grid-cols-2">
						<KeyValue label="Попытка"
							><MonoId id={attempt.id} copyKey="lab-step3-attempt" /></KeyValue
						>
						<KeyValue label="Результат">
							<StatusChip meta={ATTEMPT_RESULT[attempt.result]} raw={attempt.result} size="sm" />
						</KeyValue>
						<KeyValue label="Устройство">{shortId(attempt.device_id)}</KeyValue>
					</KeyValueGrid>
					<JsonView value={attempt} label="POST /api/v1/devices/{device?.id}/report" />
					<p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-dense text-fg-secondary">
						<a
							href="/campaigns/{campaign?.id}"
							class="underline underline-offset-4 hover:text-accent-text"
						>
							Открыть кампанию
						</a>
					</p>
				{/if}
			{/if}
		</li>
	</ol>
</Panel>
