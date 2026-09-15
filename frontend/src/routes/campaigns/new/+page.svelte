<script lang="ts">
	import { goto } from "$app/navigation";
	import { page as appPage } from "$app/state";
	import { untrack } from "svelte";
	import { createRolloutCampaign, listRolloutCampaigns } from "$lib/api/endpoints";
	import { ApiError } from "$lib/api/errors";
	import type { CampaignListItem, FirmwareVersion } from "$lib/api/types";
	import { mapCreateError, type WizardStep } from "$lib/domain/campaign-create-errors";
	import {
		coverageWarnings,
		isRowsValid,
		validateStageRows,
		type StageField,
	} from "$lib/domain/stage-draft";
	import { campaignCollection } from "$lib/state/campaign-collection.svelte";
	import { campaignDraft } from "$lib/state/campaign-draft.svelte";
	import { createResource } from "$lib/state/resource.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { toast } from "$lib/state/toast";
	import Button from "$lib/ui/Button.svelte";
	import ErrorState from "$lib/ui/ErrorState.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import DraftPreview from "$lib/screens/campaign-create/DraftPreview.svelte";
	import StepFirmware from "$lib/screens/campaign-create/StepFirmware.svelte";
	import StepHeader from "$lib/screens/campaign-create/StepHeader.svelte";
	import StepReview from "$lib/screens/campaign-create/StepReview.svelte";
	import StepStages from "$lib/screens/campaign-create/StepStages.svelte";
	import FirmwareRegisterModal from "$lib/screens/firmware/FirmwareRegisterModal.svelte";

	/**
	 * Мастер новой кампании: три шага в адресной строке, значения в общем
	 * черновике и ни одного запроса до шага 3 (backend создаёт кампанию
	 * вместе со стадиями одним POST). Шаг 1 можно открыть deep link-ом
	 * `?firmware=<uuid>`: идентификатор разрешается через индекс прошивок.
	 */
	function readStep(): WizardStep {
		const raw = Number.parseInt(appPage.url.searchParams.get("step") ?? "1", 10);
		return raw === 2 || raw === 3 ? raw : 1;
	}

	let step = $state<WizardStep>(untrack(readStep));
	let attemptedStep2 = $state(false);
	let failedSteps = $state<ReadonlySet<WizardStep>>(new Set());
	let step1FieldError = $state<string | undefined>(undefined);
	let step1RefreshFirmware = $state(false);
	let stagesBanner = $state<string | undefined>(undefined);
	let createError = $state<ApiError | undefined>(undefined);
	let registerOpen = $state(false);
	let resolvedDeepLink = $state(false);

	/** Первая страница кампаний: предупреждение о running-кампании модели. */
	const campaignsProbe = createResource<CampaignListItem[]>("campaign-create-probe", (signal) =>
		listRolloutCampaigns({ page: 1, limit: 100 }, { signal }).then(
			(response) => response.rollout_campaigns,
		),
	);

	$effect(() => {
		void firmwareIndex.ensureLoaded();
		void campaignsProbe.refresh();
	});

	// Deep link `?firmware=<uuid>`: запись из реестра подставляется в форму,
	// неизвестный или устаревший id возвращает на шаг 1 с пометкой, а не ошибкой.
	$effect(() => {
		if (resolvedDeepLink) return;
		const wanted = appPage.url.searchParams.get("firmware");
		if (!wanted) {
			resolvedDeepLink = true;
			return;
		}
		if (firmwareIndex.pending) return;
		resolvedDeepLink = true;
		const version = firmwareIndex.get(wanted);
		if (version) {
			campaignDraft.setFirmware(version.device_model, version.id);
			if (step === 1) step = 2;
		} else {
			campaignDraft.firmwareNotice = "Прошивка не найдена, выберите другую";
			step = 1;
		}
	});

	$effect(() => {
		const params = new URLSearchParams(appPage.url.searchParams);
		const current = params.get("step");
		const wanted = step === 1 ? null : String(step);
		if (current === wanted) return;
		if (wanted) params.set("step", wanted);
		else params.delete("step");
		const search = params.toString();
		void goto(`${appPage.url.pathname}${search ? `?${search}` : ""}`, {
			replaceState: true,
			keepFocus: true,
			noScroll: true,
		});
	});

	const rowErrors = $derived(validateStageRows(campaignDraft.rows));
	const step2Valid = $derived(isRowsValid(campaignDraft.rows));
	const completed = $derived.by(() => {
		const set = new Set<number>();
		if (campaignDraft.step1Complete) set.add(1);
		if (step2Valid) set.add(2);
		return set;
	});

	const runningModelCampaignId = $derived.by(() => {
		if (!campaignDraft.model) return undefined;
		return campaignsProbe.data?.find(
			(item) => item.device_model === campaignDraft.model && item.status === "running",
		)?.id;
	});

	// Предупреждение об убывании охвата показано на шаге 2, где живут строки.
	const warnings = $derived(step === 2 ? coverageWarnings(campaignDraft.rows) : []);

	// Клиентская и серверная ошибка шага 1 гаснет, как только поля заполнены.
	$effect(() => {
		if (campaignDraft.step1Complete) step1FieldError = undefined;
	});

	function canEnter(target: WizardStep): boolean {
		if (target <= step) return true;
		if (target === 2) return campaignDraft.step1Complete;
		return campaignDraft.step1Complete && step2Valid;
	}

	function enterStep(target: WizardStep): void {
		if (!canEnter(target)) return;
		step = target;
	}

	const FIELD_FOCUS: Record<StageField, string> = {
		targetPercent: "target",
		minSampleSize: "sample",
		thresholdPercent: "threshold",
	};

	function focusFirstInvalid(): void {
		for (let index = 0; index < rowErrors.length; index += 1) {
			const fields = Object.keys(rowErrors[index]) as StageField[];
			if (fields.length === 0) continue;
			const id = `stage-${index}-${FIELD_FOCUS[fields[0]]}`;
			document.getElementById(id)?.focus();
			return;
		}
	}

	function next(): void {
		if (step === 1) {
			if (!campaignDraft.step1Complete) {
				// Поля подписаны сверху, поэтому ошибка повторяет только ограничение.
				step1FieldError = step1FieldError ?? "Выберите значение";
				document.getElementById(campaignDraft.model ? "create-version" : "create-model")?.focus();
				return;
			}
			step1FieldError = undefined;
			step = 2;
			return;
		}
		if (step === 2) {
			// Вперёд только через валидацию обоих шагов: без прошивки шаг
			// сверки невозможен, поэтому возвращаем к её выбору.
			if (!campaignDraft.step1Complete) {
				step = 1;
				return;
			}
			attemptedStep2 = true;
			if (!step2Valid) {
				focusFirstInvalid();
				return;
			}
			step = 3;
		}
	}

	function clearServerErrors(): void {
		failedSteps = new Set();
		step1FieldError = undefined;
		step1RefreshFirmware = false;
		stagesBanner = undefined;
		createError = undefined;
	}

	async function create(): Promise<void> {
		if (!campaignDraft.step1Complete || !step2Valid) return;
		clearServerErrors();

		try {
			const created = await mutation.run(
				"campaign-create",
				() => createRolloutCampaign(campaignDraft.payload()),
				{ notify: false },
			);
			if (!created) return;
			const label =
				`${created.device_model} ${firmwareIndex.versionLabel(created.firmware_version_id) ?? ""}`.trimEnd();
			toast.success("Кампания создана", label, "campaign-create");
			campaignDraft.clear();
			await campaignCollection.refresh();
			await goto(`/campaigns/${created.id}`);
		} catch (error) {
			if (!(error instanceof ApiError)) return;
			if (error.status === 400) {
				const mapping = mapCreateError(error);
				failedSteps = new Set([mapping.step]);
				if (mapping.step === 1) {
					step1FieldError = mapping.fieldErrors.firmwareVersionId;
					step1RefreshFirmware = mapping.refreshFirmware ?? false;
					step = 1;
					return;
				}
				if (mapping.step === 2) {
					stagesBanner = mapping.stagesBanner;
					attemptedStep2 = true;
					step = 2;
					return;
				}
				createError = error;
				return;
			}
			createError = error;
		}
	}

	function handleRegistered(version: FirmwareVersion): void {
		campaignDraft.applyRegistered(version.device_model, version.id);
	}
</script>

<svelte:head>
	<title>Новая кампания - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="grid gap-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Новая кампания</h1>
		<div class="flex items-center gap-3">
			<StepHeader current={step} {completed} failed={failedSteps} {canEnter} onEnter={enterStep} />
			<Button
				variant="ghost"
				size="sm"
				onclick={() => {
					campaignDraft.clear();
					clearServerErrors();
					attemptedStep2 = false;
				}}
			>
				Очистить
			</Button>
		</div>
	</div>

	<div class="grid gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
		<Panel title={step === 1 ? "Прошивка" : step === 2 ? "Стадии" : "Проверка"}>
			<div aria-live="polite" class="grid gap-4">
				{#if step === 1}
					<StepFirmware
						models={firmwareIndex.models()}
						{runningModelCampaignId}
						fieldError={step1FieldError}
						refreshFirmware={step1RefreshFirmware}
						onRegister={() => (registerOpen = true)}
						onRefreshFirmware={() => {
							step1RefreshFirmware = false;
							step1FieldError = undefined;
							void firmwareIndex.refresh();
						}}
					/>
				{:else if step === 2}
					<StepStages
						attempted={attemptedStep2}
						{stagesBanner}
						{warnings}
						onApplyCanary={() => campaignDraft.applyCanaryPreset()}
						onApplySingle={() => campaignDraft.applySinglePreset()}
					/>
				{:else}
					<StepReview />
					{#if createError}
						<ErrorState error={createError} onretry={() => void create()} />
					{/if}
				{/if}

				<div
					class="flex flex-wrap items-center justify-between gap-2 border-t border-border-subtle pt-3"
				>
					<Button
						variant="secondary"
						disabled={step === 1}
						onclick={() => {
							step = step === 3 ? 2 : 1;
						}}
					>
						Назад
					</Button>

					{#if step < 3}
						<Button variant="primary" onclick={next}>Далее</Button>
					{:else}
						<Button
							variant="primary"
							loading={mutation.isPending("campaign-create")}
							disabled={!campaignDraft.step1Complete || !step2Valid}
							onclick={() => void create()}
						>
							Создать кампанию
						</Button>
					{/if}
				</div>
			</div>
		</Panel>

		<div class="lg:sticky lg:top-[72px] lg:self-start">
			<DraftPreview />
		</div>
	</div>
</div>

<FirmwareRegisterModal
	open={registerOpen}
	onOpenChange={(open) => (registerOpen = open)}
	onCreated={handleRegistered}
/>
