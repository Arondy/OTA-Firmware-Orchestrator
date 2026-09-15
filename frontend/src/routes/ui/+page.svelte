<script lang="ts">
	import FlaskIcon from "phosphor-svelte/lib/FlaskIcon";
	import RocketIcon from "phosphor-svelte/lib/RocketIcon";
	import PlusIcon from "phosphor-svelte/lib/PlusIcon";
	import TrashIcon from "phosphor-svelte/lib/TrashIcon";
	import { ApiError } from "$lib/api/errors";
	import { listDevices } from "$lib/api/endpoints";
	import type { Device, Stage } from "$lib/api/types";
	import { createPagedResource } from "$lib/state/paged-resource.svelte";
	import { ATTEMPT_RESULT, CAMPAIGN_STATUS, DEVICE_STATUS, STAGE_STATUS } from "$lib/domain/status";
	import Button from "$lib/ui/Button.svelte";
	import Checkbox from "$lib/ui/Checkbox.svelte";
	import ConfirmDialog from "$lib/ui/ConfirmDialog.svelte";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import DataTable from "$lib/ui/DataTable.svelte";
	import type { Column } from "$lib/ui/table-types";
	import Drawer from "$lib/ui/Drawer.svelte";
	import EmptyState from "$lib/ui/EmptyState.svelte";
	import ErrorState from "$lib/ui/ErrorState.svelte";
	import ExternalLink from "$lib/ui/ExternalLink.svelte";
	import Field from "$lib/ui/Field.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import IndeterminateBar from "$lib/ui/IndeterminateBar.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import JsonView from "$lib/ui/JsonView.svelte";
	import Kbd from "$lib/ui/Kbd.svelte";
	import KeyValue from "$lib/ui/KeyValue.svelte";
	import KeyValueGrid from "$lib/ui/KeyValueGrid.svelte";
	import LiveDot from "$lib/ui/LiveDot.svelte";
	import Metric from "$lib/ui/Metric.svelte";
	import Modal from "$lib/ui/Modal.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import NumberInput from "$lib/ui/NumberInput.svelte";
	import Pagination from "$lib/ui/Pagination.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import Progress from "$lib/ui/Progress.svelte";
	import RadioGroup from "$lib/ui/RadioGroup.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import SegmentedControl from "$lib/ui/SegmentedControl.svelte";
	import Select from "$lib/ui/Select.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import Spinner from "$lib/ui/Spinner.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Switch from "$lib/ui/Switch.svelte";
	import Tabs from "$lib/ui/Tabs.svelte";
	import Textarea from "$lib/ui/Textarea.svelte";
	import TextInput from "$lib/ui/TextInput.svelte";
	import ThemeToggle from "$lib/ui/ThemeToggle.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import Gauge from "$lib/viz/Gauge.svelte";
	import SampleProgress from "$lib/viz/SampleProgress.svelte";
	import StagePipeline from "$lib/viz/StagePipeline.svelte";
	import StackedBar from "$lib/viz/StackedBar.svelte";

	// Значения этого экрана - fixtures стилей, а не данные системы.
	const demoError = new ApiError({
		status: 503,
		method: "POST",
		url: "/api/v1/devices/3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c/report",
		requestId: "demo-request-id",
		serverMessage: "failed to produce update result",
	});

	const demoStages: Stage[] = [
		{
			id: "demo-stage-1",
			campaign_id: "demo",
			order_index: 0,
			target_percent: 10,
			min_sample_size: 20,
			success_threshold: 0.95,
			status: "passed",
			entered_at: "2026-09-12T08:00:00.000Z",
		},
		{
			id: "demo-stage-2",
			campaign_id: "demo",
			order_index: 1,
			target_percent: 50,
			min_sample_size: 20,
			success_threshold: 0.95,
			status: "active",
			entered_at: "2026-09-12T09:30:00.000Z",
		},
		{
			id: "demo-stage-3",
			campaign_id: "demo",
			order_index: 2,
			target_percent: 100,
			min_sample_size: 20,
			success_threshold: 0.95,
			status: "pending",
		},
	];

	const demoCounts = { draft: 3, running: 12, paused: 4, completed: 30, rolled_back: 6 };

	let textValue = $state("demo-sensor-v1");
	let invalidValue = $state("не semver");
	let numberValue = $state<number | undefined>(10);
	let areaValue = $state("Описание раскатки");
	let selectValue = $state("running");
	let checked = $state(true);
	let switched = $state(false);
	let radioValue = $state("a");
	let segmentValue = $state("50");
	let modalOpen = $state(false);
	let confirmOpen = $state(false);
	let drawerOpen = $state(false);
	let metricValue = $state(42);
	let tabValue = $state("one");

	const devices = createPagedResource<Device, Record<string, string | undefined>>({
		key: "ui-devices",
		fetch: ({ page, limit, filters, signal }) =>
			listDevices({ page, limit, ...filters }, { signal }).then((response) => response.devices),
		initialFilters: {},
	});

	const columns: Column<Device>[] = [
		{ id: "id", header: "устройство", sortValue: (row) => row.id },
		{ id: "model", header: "модель", sortValue: (row) => row.device_model },
		{ id: "version", header: "версия", align: "numeric", sortValue: (row) => row.current_version },
		{ id: "status", header: "статус" },
		{ id: "seen", header: "последняя отметка", align: "right" },
	];
</script>

{#snippet deviceCell(id: string, row: Device)}
	{#if id === "id"}
		<MonoId id={row.id} copyKey={`ui-${row.id}`} />
	{:else if id === "model"}
		<span class="font-mono text-table text-fg-primary">{row.device_model}</span>
	{:else if id === "version"}
		<span>{row.current_version}</span>
	{:else if id === "status"}
		<StatusChip meta={DEVICE_STATUS[row.status]} raw={row.status} size="sm" />
	{:else}
		<RelativeTime iso={row.last_seen} />
	{/if}
{/snippet}

<svelte:head>
	<title>Стилевод - OTA Firmware Orchestrator</title>
</svelte:head>

<div class="flex flex-wrap items-center justify-between gap-3">
	<div>
		<h1 class="text-page font-semibold tracking-[-0.01em] text-fg-primary">Стилевод</h1>
		<p class="mt-1 max-w-[70ch] text-table text-fg-secondary">
			Служебный экран: витрина примитивов и их состояний для визуальных ревизий. Значения ниже -
			fixtures стилей, а не данные системы. В навигации экрана нет.
		</p>
	</div>
	<ThemeToggle />
</div>

<div class="mt-6 grid gap-4">
	<Panel title="Кнопки">
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="primary" icon={PlusIcon}>Создать</Button>
			<Button variant="secondary">Отменить</Button>
			<Button variant="ghost">Сбросить</Button>
			<Button variant="danger" icon={TrashIcon}>Откатить</Button>
			<Button variant="primary" size="sm">Маленькая</Button>
			<Button variant="primary" loading>Загрузка</Button>
			<Button variant="primary" disabled>Недоступна</Button>
			<IconButton glyph={PlusIcon} ariaLabel="Добавить" />
			<IconButton glyph={TrashIcon} ariaLabel="Удалить" variant="danger" />
		</div>
	</Panel>

	<Panel title="Статусы">
		<div class="flex flex-wrap gap-2">
			{#each Object.entries(CAMPAIGN_STATUS) as [value, meta] (value)}
				<StatusChip {meta} raw={value} />
			{/each}
		</div>
		<div class="mt-2 flex flex-wrap gap-2">
			{#each Object.entries(STAGE_STATUS) as [value, meta] (value)}
				<StatusChip {meta} raw={value} size="sm" />
			{/each}
			{#each Object.entries(ATTEMPT_RESULT) as [value, meta] (value)}
				<StatusChip {meta} raw={value} size="sm" />
			{/each}
		</div>
		<p class="mt-3 flex items-center gap-2 text-dense text-fg-secondary">
			<LiveDot class="bg-accent" /> живой индикатор пульсирует, при reduced motion стоит
		</p>
	</Panel>

	<Panel title="Формы">
		<div class="grid gap-4 md:grid-cols-2">
			<Field label="Модель устройства" for="ui-model" hint="Точное совпадение, 2-64 символа">
				<TextInput id="ui-model" bind:value={textValue} />
			</Field>
			<Field label="Версия прошивки" for="ui-version" error="Field must be a valid semver string">
				<TextInput id="ui-version" bind:value={invalidValue} mono />
			</Field>
			<Field label="Охват стадии" for="ui-percent">
				<NumberInput id="ui-percent" bind:value={numberValue} min={1} max={100}>
					{#snippet unit()}%{/snippet}
				</NumberInput>
			</Field>
			<Field label="Статус кампании">
				<Select
					id="ui-status"
					bind:value={selectValue}
					ariaLabel="Статус кампании"
					items={Object.entries(CAMPAIGN_STATUS).map(([value, meta]) => ({
						value,
						label: meta.label,
					}))}
				/>
			</Field>
			<Field label="Комментарий" for="ui-note">
				<Textarea id="ui-note" bind:value={areaValue} />
			</Field>
			<div class="grid content-start gap-2">
				<Checkbox id="ui-check" bind:checked label="Показывать завершённые" />
				<Switch id="ui-switch" bind:checked={switched} label="Плотная таблица" />
				<RadioGroup
					name="ui-radio"
					bind:value={radioValue}
					legend="Режим отката"
					items={[
						{ value: "a", label: "ручной" },
						{ value: "b", label: "автоматический" },
					]}
				/>
				<SegmentedControl
					name="ui-segment"
					value={segmentValue}
					ariaLabel="Охват"
					items={[
						{ value: "10", label: "10%" },
						{ value: "50", label: "50%" },
						{ value: "100", label: "100%" },
					]}
					onchange={(value) => (segmentValue = value)}
				/>
			</div>
		</div>
	</Panel>

	<Panel title="Оверлеи и подсказки">
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="secondary" onclick={() => (modalOpen = true)}>Открыть диалог</Button>
			<Button variant="danger" onclick={() => (confirmOpen = true)}>Запросить подтверждение</Button>
			<Button variant="secondary" onclick={() => (drawerOpen = true)}>Открыть drawer</Button>
			<Tooltip label="Подсказка появляется с задержкой 1 с">
				<Button variant="ghost">Навести курсор</Button>
			</Tooltip>
			<span class="text-table text-fg-secondary">клавиша <Kbd>/</Kbd></span>
		</div>
	</Panel>

	<Panel title="Обратная связь">
		<div class="grid gap-3">
			<div class="flex flex-wrap items-center gap-3">
				<Spinner />
				<Skeleton shape="chip" />
				<Skeleton shape="text" rows={2} class="max-w-72" />
				<Skeleton shape="metric" />
			</div>
			<IndeterminateBar label="Выполняется асинхронное действие" />
			<InlineBanner tone="info">Информационное сообщение панели.</InlineBanner>
			<InlineBanner tone="warning">Кампания близка к порогу успеха.</InlineBanner>
			<InlineBanner tone="danger">Оркестратор не отвечает.</InlineBanner>
			<ErrorState error={demoError} />
			<EmptyState
				icon={FlaskIcon}
				title="Песочница пуста"
				text="Запустите симулятор устройства, чтобы увидеть чек-ины и отчёты."
			>
				{#snippet action()}
					<Button variant="primary" icon={PlusIcon}>Запустить</Button>
				{/snippet}
			</EmptyState>
		</div>
	</Panel>

	<Panel title="Показатели">
		<div class="grid gap-4 md:grid-cols-3">
			<Metric
				label="устройств на связи"
				value={String(metricValue)}
				hint="нажмите, чтобы изменить"
			/>
			<Button variant="secondary" size="sm" onclick={() => (metricValue += 7)}>
				Изменить значение
			</Button>
			<Progress value={0.62} label="охват активной стадии" />
		</div>
		<KeyValueGrid class="mt-4">
			<KeyValue label="модель">demo-sensor-v1</KeyValue>
			<KeyValue label="целевая версия" mono>2.0.0</KeyValue>
			<KeyValue label="создана"><RelativeTime iso="2026-09-12T08:00:00.000Z" /></KeyValue>
			<KeyValue label="идентификатор">
				<CopyButton
					text="3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c"
					copyKey="ui-copy"
					display="3f6d2d8e…a3b4c"
				/>
			</KeyValue>
		</KeyValueGrid>
		<div class="mt-3 flex flex-wrap items-center gap-3 text-table">
			<ExternalLink href="https://svelte.dev">Документация Svelte</ExternalLink>
			<MonoId id="3f6d2d8e-4b1c-4a5f-9a7b-8e7d1c2a3b4c" copyKey="ui-monoid" />
		</div>
		<div class="mt-3">
			<JsonView
				label="ответ /healthz и пример тела"
				value={{ status: "OK", success_rate: 0.9849, sample_size: 128, paused: false }}
			/>
		</div>
		<div class="mt-3">
			<Tabs
				bind:value={tabValue}
				ariaLabel="Пример вкладок"
				items={[
					{ value: "one", label: "Первая" },
					{ value: "two", label: "Вторая" },
				]}
			>
				{#snippet panel(active: string)}
					<p class="text-table text-fg-secondary">Содержимое вкладки: {active}</p>
				{/snippet}
			</Tabs>
		</div>
	</Panel>

	<Panel title="Визуализации">
		<div class="grid gap-4 md:grid-cols-2">
			<div class="grid gap-2">
				<StagePipeline stages={demoStages} />
				<StagePipeline stages={demoStages} mode="compact" />
			</div>
			<Gauge value={0.91} threshold={0.95} />
			<Gauge value={0.98} threshold={0.95} />
			<Gauge value={undefined} threshold={0.95} />
			<SampleProgress sampleSize={12} minSampleSize={20} />
			<SampleProgress sampleSize={24} minSampleSize={20} />
		</div>
		<div class="mt-4">
			<StackedBar counts={demoCounts} />
		</div>
	</Panel>

	<Panel title="Таблица и пагинация (реальные данные)">
		<DataTable
			{columns}
			rows={devices.rows}
			rowKey={(row) => row.id}
			cell={deviceCell}
			caption="Устройства из первой загруженной страницы"
			loading={devices.pending}
			error={devices.error}
			onRetry={() => devices.refresh()}
		>
			{#snippet emptySnippet()}
				<EmptyState
					icon={RocketIcon}
					title="Устройств пока нет"
					text="Зарегистрируйте устройство, чтобы увидеть его в таблице."
				/>
			{/snippet}
		</DataTable>
		<div class="mt-3">
			<Pagination
				page={devices.page}
				limit={devices.limit}
				hasMore={devices.hasMore}
				onPage={(page) => devices.setPage(page)}
				onLimit={(limit) => devices.setLimit(limit)}
			/>
		</div>
	</Panel>
</div>

<Modal
	open={modalOpen}
	onOpenChange={(open) => (modalOpen = open)}
	title="Пример диалога"
	description="Фокус удерживается внутри, Esc и клик по фону закрывают."
>
	<p class="text-table text-fg-secondary">Содержимое диалога.</p>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (modalOpen = false)}>Закрыть</Button>
	{/snippet}
</Modal>

<ConfirmDialog
	open={confirmOpen}
	onOpenChange={(open) => (confirmOpen = open)}
	title="Откатить кампанию?"
	body="Действие необратимо: устройства вернутся на предыдущую версию прошивки."
	confirmLabel="Откатить"
	tone="danger"
	onconfirm={() => (confirmOpen = false)}
/>

<Drawer open={drawerOpen} onOpenChange={(open) => (drawerOpen = open)} label="Пример drawer">
	<div class="p-4 text-table text-fg-secondary">Так выглядит мобильное меню разделов.</div>
</Drawer>
