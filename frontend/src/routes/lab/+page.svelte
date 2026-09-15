<script lang="ts">
	import { goto } from "$app/navigation";
	import { page as appPage } from "$app/state";
	import { untrack } from "svelte";
	import { createDevice, getRolloutCampaign, listDevices } from "$lib/api/endpoints";
	import type { Campaign, Device } from "$lib/api/types";
	import { stageNumber } from "$lib/domain/campaign-history";
	import { campaignCollection } from "$lib/state/campaign-collection.svelte";
	import { campaignDetails } from "$lib/state/campaign-details.svelte";
	import { createResource } from "$lib/state/resource.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { batchRunner } from "$lib/state/batch-runner.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { ApiError } from "$lib/api/errors";
	import ActionButton from "$lib/ui/ActionButton.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import Field from "$lib/ui/Field.svelte";
	import RocketIcon from "phosphor-svelte/lib/RocketIcon";
	import SearchSelect from "$lib/ui/SearchSelect.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import LabBatchPanel from "$lib/screens/lab/LabBatchPanel.svelte";
	import LabDevicePanel from "$lib/screens/lab/LabDevicePanel.svelte";
	import LabLogPanel from "$lib/screens/lab/LabLogPanel.svelte";
	import LabStepsPanel from "$lib/screens/lab/LabStepsPanel.svelte";

	/**
	 * Песочница: симулятор устройства поверх настоящего API. Кампания-цель
	 * выбирается в шапке и живёт в URL (`?campaign=`), deep link `?device=`
	 * подставляет устройство с первой страницы реестра модели.
	 */
	const NO_CAMPAIGN = "";

	let targetId = $state(untrack(() => appPage.url.searchParams.get("campaign") ?? NO_CAMPAIGN));
	let device = $state<Device | undefined>(undefined);
	let registerResult = $state<Device | undefined>(undefined);
	let registerModel = $state("");
	let registerVersion = $state("");
	let deviceDeepLink = $state(untrack(() => appPage.url.searchParams.get("device") ?? ""));

	$effect(() => {
		untrack(() => {
			void firmwareIndex.ensureLoaded();
			if (campaignCollection.lastUpdatedAt === undefined && !campaignCollection.walking) {
				void campaignCollection.refresh();
			}
		});
	});

	const runningCampaigns = $derived(
		campaignCollection.rows.filter((row) => row.status === "running"),
	);

	// Подписи селектора берут стадии из агрегатора деталей: список кампаний
	// стадий не несёт, а оператору виден номер и охват активной.
	$effect(() => {
		campaignDetails.ensure(runningCampaigns.map((row) => row.id));
	});

	const campaign = createResource<Campaign | undefined>("lab-target-campaign", (signal) =>
		targetId ? getRolloutCampaign(targetId, { signal }) : Promise.resolve(undefined),
	);

	let lastTarget = $state<string | null>(null);
	$effect(() => {
		if (lastTarget === targetId) return;
		lastTarget = targetId;
		device = undefined;
		registerResult = undefined;
		void campaign.refresh();
	});

	const targetModel = $derived(campaign.data?.device_model ?? "");
	const activeStage = $derived(
		campaign.data?.rollout_stages.find((stage) => stage.status === "active"),
	);
	const targetVersion = $derived(firmwareIndex.versionLabel(campaign.data?.firmware_version_id));

	const devices = createResource<Device[]>("lab-target-devices", (signal) =>
		targetModel
			? listDevices({ device_model: targetModel, limit: 100 }, { signal }).then(
					(response) => response.devices,
				)
			: Promise.resolve([]),
	);

	let lastModel = $state<string | null>(null);
	$effect(() => {
		if (lastModel === targetModel) return;
		lastModel = targetModel;
		void devices.refresh();
	});

	// Deep link устройства: свежая регистрация всегда на первой странице.
	$effect(() => {
		if (!deviceDeepLink || device || devices.pending) return;
		const found = devices.data?.find((row) => row.id === deviceDeepLink);
		if (found) device = found;
		deviceDeepLink = "";
	});

	$effect(() => {
		const params = new URLSearchParams(appPage.url.searchParams);
		const current = params.get("campaign");
		const wanted = targetId === NO_CAMPAIGN ? null : targetId;
		if (current === wanted) return;
		if (wanted) params.set("campaign", wanted);
		else params.delete("campaign");
		params.delete("device");
		const search = params.toString();
		void goto(`${appPage.url.pathname}${search ? `?${search}` : ""}`, {
			replaceState: true,
			keepFocus: true,
			noScroll: true,
		});
	});

	// Guard: закрытие вкладки посреди прогона рвёт запросы и теряет журнал.
	$effect(() => {
		if (!batchRunner.running) return;
		const handler = (event: BeforeUnloadEvent) => {
			event.preventDefault();
		};
		window.addEventListener("beforeunload", handler);
		return () => window.removeEventListener("beforeunload", handler);
	});

	// Единый элемент выбора: тот же SearchSelect, что на остальных экранах.
	// Пустое значение - «без кампании», отдельного варианта в списке нет.
	const targetItems = $derived(
		runningCampaigns.map((row) => {
			const version = firmwareIndex.versionLabel(row.firmware_version_id);
			const detail = campaignDetails.get(row.id);
			const stage = detail?.rollout_stages.find((item) => item.status === "active");
			const stageLabel = stage
				? `, стадия ${stageNumber(stage.order_index)} (${stage.target_percent}%)`
				: "";
			return {
				value: row.id,
				label: `${row.device_model}, ${version ?? "версия неизвестна"}${stageLabel}`,
			};
		}),
	);

	async function registerDevice(model: string, version: string): Promise<Device | undefined> {
		try {
			const created = await mutation.run("lab-register", () =>
				createDevice({ device_model: model, current_version: version }),
			);
			if (!created) return undefined;
			batchRunner.noteManual(
				"register",
				created.id,
				201,
				`модель ${model}, версия ${version}`,
				"neutral",
			);
			device = created;
			registerResult = created;
			await devices.refresh();
			return created;
		} catch (error) {
			const status = error instanceof ApiError ? error.status : undefined;
			const message =
				error instanceof ApiError ? (error.serverMessage ?? error.headline) : String(error);
			batchRunner.noteManual("register", undefined, status, `сбой: ${message}`, "danger");
			return undefined;
		}
	}

	// Чек-ин на сервере обновляет last_seen: отражаем это сразу в выбранном
	// устройстве и в списке, чтобы панель не показывала устаревшее нет отметки.
	function noteCheckin(): void {
		if (!device) return;
		const at = new Date().toISOString();
		device = { ...device, last_seen: at };
		const list = devices.data;
		if (list) {
			devices.write(list.map((row) => (row.id === device?.id ? { ...row, last_seen: at } : row)));
		}
	}
</script>

<svelte:head>
	<title>Песочница - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="grid gap-4">
	<div class="grid gap-1">
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Песочница</h1>
	</div>

	<Field label="Кампания" class="max-w-md">
		<SearchSelect
			id="lab-target"
			bind:value={targetId}
			items={targetItems}
			ariaLabel="Кампания"
			placeholder="без кампании"
		/>
	</Field>

	{#if campaignCollection.lastUpdatedAt !== undefined && runningCampaigns.length === 0}
		<EmptyState
			icon={RocketIcon}
			title="Нет выполняющихся кампаний"
			text="Песочнице нужна выполняющаяся кампания: создайте и запустите её."
		>
			{#snippet action()}
				<ActionButton href="/campaigns/new">Новая кампания</ActionButton>
			{/snippet}
		</EmptyState>
	{/if}

	<div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
		<div class="grid content-start gap-5">
			{#if campaign.pending}
				<div class="grid gap-3 rounded-panel border border-border-subtle bg-bg-surface p-4">
					<Skeleton class="h-5 w-40" />
					<Skeleton class="h-9 w-full" />
					<Skeleton class="h-9 w-full" />
					<Skeleton class="h-24 w-full" />
				</div>
			{:else}
				<LabDevicePanel
					devices={devices.data ?? []}
					{device}
					campaignId={campaign.data?.id}
					campaignModel={targetModel}
					{targetVersion}
					targetPercent={activeStage?.target_percent}
					bind:registerModel
					bind:registerVersion
					onPick={(picked) => (device = picked)}
					onRegister={registerDevice}
				/>
				<LabStepsPanel
					{device}
					campaign={campaign.data}
					{registerResult}
					onRegister={() => registerDevice(registerModel.trim(), registerVersion.trim())}
					onCheckin={noteCheckin}
				/>
			{/if}
		</div>

		<LabLogPanel lines={batchRunner.lines} />
	</div>

	<LabBatchPanel
		campaign={campaign.data}
		runningCampaignForModel={(model) => {
			const item = runningCampaigns.find((row) => row.device_model === model);
			return item ? campaignDetails.get(item.id) : undefined;
		}}
	/>
</div>
