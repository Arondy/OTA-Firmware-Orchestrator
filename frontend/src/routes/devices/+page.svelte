<script lang="ts">
	import { untrack } from "svelte";
	import CpuIcon from "phosphor-svelte/lib/CpuIcon";
	import { decommissionDevice, listDevices } from "$lib/api/endpoints";
	import { ApiError } from "$lib/api/errors";
	import type { Device } from "$lib/api/types";
	import { apiAvailability } from "$lib/state/api-availability.svelte";
	import { campaignCollection } from "$lib/state/campaign-collection.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { mutation } from "$lib/state/mutation.svelte";
	import { toast } from "$lib/state/toast";
	import {
		createPagedResource,
		type Filters,
		type PagedResource,
	} from "$lib/state/paged-resource.svelte";
	import ActionButton from "$lib/ui/ActionButton.svelte";
	import Button from "$lib/ui/Button.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import ConfirmDialog from "$lib/ui/ConfirmDialog.svelte";
	import Pagination from "$lib/ui/Pagination.svelte";
	import DeviceRegisterModal from "$lib/screens/devices/DeviceRegisterModal.svelte";
	import DevicesFilters from "$lib/screens/devices/DevicesFilters.svelte";
	import DevicesTable from "$lib/screens/devices/DevicesTable.svelte";

	/**
	 * Устройства: самый большой набор данных системы, поэтому серверные
	 * фильтры и пагинация без общего числа - основа экрана (§5.2). Быстрый
	 * чип «нет отметки» считает только загруженную страницу и не претендует
	 * на глобальность: фильтра по last_seen у API нет.
	 */
	interface DeviceFilters extends Filters {
		model: string;
		status: string;
	}

	const resource: PagedResource<Device, DeviceFilters> = createPagedResource({
		key: "devices",
		fetch: ({ page, limit, filters, signal }) =>
			listDevices(
				{
					page,
					limit,
					// Пустые значения не отправляются вовсе: `device_model=` дал бы 400.
					device_model: filters.model || undefined,
					status:
						filters.status === "active" || filters.status === "decommissioned"
							? filters.status
							: undefined,
				},
				{ signal },
			).then((response) => response.devices),
		initialFilters: { model: "", status: "" },
		limit: 50,
	});

	let modelFilter = $state(resource.filters.model ?? "");
	let statusFilter = $state(resource.filters.status ?? "");
	let noMarkActive = $state(false);
	let registerOpen = $state(false);
	let confirmDevice = $state<Device | undefined>(undefined);
	let actionError = $state<string | undefined>(undefined);

	$effect(() => {
		// Разовая подтяжка индексов: реактивные чтения внутри untrack, иначе
		// смена walking/lastUpdatedAt перезапускала бы эффект и запросы.
		untrack(() => {
			void firmwareIndex.ensureLoaded();
			if (campaignCollection.lastUpdatedAt === undefined && !campaignCollection.walking) {
				void campaignCollection.refresh();
			}
		});
	});

	// Локальные значения фильтров и состояние ресурса сходятся в одну точку:
	// адресную строку пишет постаничный ресурс.
	$effect(() => {
		const model = resource.filters.model ?? "";
		const status = resource.filters.status ?? "";
		if (modelFilter === model && statusFilter === status) return;
		void resource.setFilters({ model: modelFilter, status: statusFilter });
	});
	$effect(() => {
		const model = resource.filters.model ?? "";
		const status = resource.filters.status ?? "";
		if (model !== modelFilter) modelFilter = model;
		if (status !== statusFilter) statusFilter = status;
	});

	const noMarkCount = $derived(resource.rows.filter((row) => !row.last_seen).length);
	const shownRows = $derived(
		noMarkActive ? resource.rows.filter((row) => !row.last_seen) : resource.rows,
	);
	const filtersActive = $derived(modelFilter !== "" || statusFilter !== "" || noMarkActive);

	const modelOptions = $derived.by(() => {
		const models = new Set(firmwareIndex.models());
		for (const row of resource.rows) models.add(row.device_model);
		if (modelFilter) models.add(modelFilter);
		return [...models].sort((a, b) => a.localeCompare(b, "ru"));
	});

	/** Целевая версия выполняющейся кампании модели: для чипа «целевая». */
	const runningTargetByModel = $derived.by(() => {
		const map = new Map<string, string>();
		for (const campaign of campaignCollection.rows) {
			if (campaign.status !== "running") continue;
			const version = firmwareIndex.versionLabel(campaign.firmware_version_id);
			if (version) map.set(campaign.device_model, version);
		}
		return map;
	});

	const registryEmpty = $derived(
		!resource.pending && resource.rows.length === 0 && !filtersActive && !resource.error,
	);

	async function decommissionConfirmed(): Promise<void> {
		const device = confirmDevice;
		if (!device) return;
		actionError = undefined;

		try {
			const updated = await mutation.run(`device-decommission-${device.id}`, () =>
				decommissionDevice(device.id),
			);
			if (!updated) return;
			// Ответ и есть обновлённое устройство: латка строки без перезапроса.
			resource.update((rows) => rows.map((row) => (row.id === updated.id ? updated : row)));
			toast.success(
				"Устройство выведено из эксплуатации",
				updated.device_model,
				`device-${updated.id}`,
			);
		} catch (error) {
			if (!(error instanceof ApiError)) return;
			if (error.status === 404) {
				toast.error("Устройство не найдено", error.hint, `device-${device.id}`);
				await resource.refresh();
				return;
			}
			actionError = error.headline;
		} finally {
			confirmDevice = undefined;
		}
	}
</script>

<svelte:head>
	<title>Устройства - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="grid gap-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Устройства</h1>
		<ActionButton onclick={() => (registerOpen = true)}>Зарегистрировать устройство</ActionButton>
	</div>

	<DevicesFilters
		bind:model={modelFilter}
		status={statusFilter}
		models={modelOptions}
		{noMarkCount}
		{noMarkActive}
		{filtersActive}
		onStatus={(status) => {
			statusFilter = status;
		}}
		onNoMark={() => (noMarkActive = !noMarkActive)}
		onReset={() => {
			noMarkActive = false;
			modelFilter = "";
			statusFilter = "";
			void resource.resetFilters();
		}}
	/>

	<!-- Один сбой - одно сообщение: сетевые отказы уже объявлены глобальной
        	полосой «API недоступен», экранный баннер остаётся за сбоями ответа
        	(400, 500), а при пустой таблице ошибку показывает сама таблица. -->
	{#if resource.error && !apiAvailability.down && resource.rows.length > 0}
		<InlineBanner tone="danger" boxed>
			<span class="flex flex-wrap items-center gap-x-3 gap-y-1">
				{resource.error.headline}
				<Button variant="secondary" size="sm" onclick={() => void resource.refresh()}>
					Повторить
				</Button>
			</span>
		</InlineBanner>
	{/if}

	{#if actionError}
		<InlineBanner tone="danger" boxed>{actionError}</InlineBanner>
	{/if}

	{#if registryEmpty}
		<EmptyState
			icon={CpuIcon}
			title="Устройств пока нет"
			text="Зарегистрируйте первое устройство или запустите сид tests/seed."
		>
			{#snippet action()}
				<ActionButton onclick={() => (registerOpen = true)}>
					Зарегистрировать устройство
				</ActionButton>
			{/snippet}
		</EmptyState>
	{:else if !resource.pending && shownRows.length === 0 && filtersActive}
		<EmptyState icon={CpuIcon} title="Ничего не найдено">
			{#snippet action()}
				<Button
					variant="secondary"
					onclick={() => {
						noMarkActive = false;
						modelFilter = "";
						statusFilter = "";
						void resource.resetFilters();
					}}
				>
					Сбросить фильтры
				</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<DevicesTable
			rows={shownRows}
			loading={resource.pending}
			runningTargetVersionByModel={runningTargetByModel}
			skeletonRows={resource.limit}
			emptyLabel={resource.error
				? apiAvailability.down
					? "Данные не загружены"
					: "Не удалось загрузить данные"
				: "Устройств нет"}
			error={resource.error && !apiAvailability.down && shownRows.length === 0
				? resource.error
				: undefined}
			onRetry={() => void resource.refresh()}
			onModelFilter={(model) => {
				modelFilter = model;
			}}
			onDecommission={(device) => (confirmDevice = device)}
		/>
	{/if}

	<div class="grid gap-2">
		<Pagination
			page={resource.page}
			limit={resource.limit}
			hasMore={resource.hasMore}
			onPage={(page) => void resource.setPage(page)}
			onLimit={(limit) => void resource.setLimit(limit)}
		/>
	</div>
</div>

<ConfirmDialog
	open={confirmDevice !== undefined}
	onOpenChange={(open) => {
		if (!open) confirmDevice = undefined;
	}}
	title="Вывести устройство из эксплуатации?"
	body="Устройство перестанет получать обновления. Действие нельзя отменить."
	confirmLabel="Вывести"
	tone="danger"
	busy={confirmDevice ? mutation.isPending(`device-decommission-${confirmDevice.id}`) : false}
	onconfirm={() => void decommissionConfirmed()}
/>

<DeviceRegisterModal
	open={registerOpen}
	onOpenChange={(open) => (registerOpen = open)}
	onCreated={() => {
		noMarkActive = false;
		void resource.setPage(1);
	}}
/>
