<script lang="ts">
	import PowerIcon from "phosphor-svelte/lib/PowerIcon";
	import type { ApiError } from "$lib/api/errors";
	import type { Device } from "$lib/api/types";
	import {
		isOnTargetVersion,
		lastSeenMarker,
		MARKER_DOT,
		MARKER_TITLE,
		TARGET_CHIP_TITLE,
	} from "$lib/domain/device-row";
	import { DEVICE_STATUS, NO_LAST_SEEN_LABEL, NO_LAST_SEEN_TITLE } from "$lib/domain/status";
	import { clock } from "$lib/state/clock.svelte";
	import DataTable from "$lib/ui/DataTable.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import type { Column } from "$lib/ui/table-types";
	import { cn } from "$lib/ui/cn";

	/**
	 * Таблица устройств: серверные данные без клиентских домыслов. Отсутствие
	 * `last_seen` - это «нет отметки» с пояснением про TTL Redis, а не
	 * «офлайн»; выведенные строки приглушены подписью, а не зачёркиванием.
	 * На узком экране таблица становится списком карточек (§6).
	 */
	let {
		rows,
		loading,
		runningTargetVersionByModel,
		emptyLabel = "Устройств нет",
		skeletonRows = 25,
		error,
		onRetry,
		onModelFilter,
		onDecommission,
	}: {
		rows: Device[];
		loading: boolean;
		/** Целевая версия выполняющейся кампании по модели устройства. */
		runningTargetVersionByModel: ReadonlyMap<string, string>;
		/** Подпись пустой области: зависит от того, ошибка ли это или пустая база. */
		emptyLabel?: string;
		skeletonRows?: number;
		error?: ApiError;
		onRetry?: () => void;
		onModelFilter: (model: string) => void;
		onDecommission: (device: Device) => void;
	} = $props();

	// Ширины в px зафиксированы и растягиваются драгом разделителей:
	// раскладка не зависит от содержимого строк и активных фильтров.
	// «Действия» держат ту же ширину, что в таблице кампаний: колонка
	// одинаковая во всех списках, поэтому кнопки не «прыгают» между экранами.
	const columns: Column<Device>[] = [
		{ id: "device", header: "Устройство", width: 160 },
		{ id: "model", header: "Модель", width: 176 },
		{ id: "version", header: "Версия", width: 144 },
		{ id: "status", header: "Статус", width: 144 },
		{ id: "lastSeen", header: "Последняя отметка", align: "right", width: 150 },
		{ id: "created", header: "Зарегистрировано", align: "right", width: 130 },
		{ id: "actions", header: "Действия", align: "right", width: 112 },
	];

	function markerOf(device: Device) {
		return lastSeenMarker(device.last_seen, clock.now);
	}

	/**
	 * Есть ли что показать в подписи связности **в карточке**. У выведенного из
	 * эксплуатации устройства пустой отметки не бывает по смыслу: оно снято с
	 * раскаток и молчит, поэтому «нет отметки» рядом с ним читалось бы как
	 * неполадка, а не как следствие статуса. Реальную отметку, если она ещё
	 * жива в Redis, показываем и у выведенного - это факт, а не домысел.
	 *
	 * В таблице это правило не действует: у колонки «Последняя отметка» есть
	 * собственный заголовок, и пустая ячейка там читается не как «отметки нет
	 * по статусу», а как недогруженные данные.
	 */
	function cardShowsLiveness(device: Device) {
		return Boolean(device.last_seen) || device.status !== "decommissioned";
	}
</script>

<!-- Узкий экран: карточки вместо таблицы. -->
<ol class="grid gap-2 md:hidden">
	{#if loading}
		{#each Array.from({ length: 6 }) as _, index (index)}
			<li class="grid gap-2 rounded-panel border border-border-subtle bg-bg-surface p-3">
				<Skeleton class="h-4 w-2/3" />
				<Skeleton class="h-4 w-1/2" />
				<Skeleton class="h-8 w-full" />
			</li>
		{/each}
	{/if}
	{#each rows as device (device.id)}
		{@const marker = markerOf(device)}
		<li
			class={cn(
				"grid gap-1.5 rounded-panel border border-border-subtle bg-bg-surface p-3",
				device.status === "decommissioned" && "text-fg-muted",
			)}
		>
			<!-- Правая подпись стоит на уровне названия модели, а не отдельной
			     строкой под идентификатором: она отвечает на вопрос «как давно
			     устройство на связи», и читать её рядом с названием дешевле. -->
			<div class="flex flex-wrap items-center gap-2">
				<span class="text-table font-medium text-fg-primary">{device.device_model}</span>
				<span class="font-mono text-table tabular-nums text-fg-secondary">
					{device.current_version}
				</span>
				<StatusChip meta={DEVICE_STATUS[device.status]} raw={device.status} size="sm" />
				{#if cardShowsLiveness(device)}
					<span class="ml-auto flex items-center gap-1.5 text-dense text-fg-secondary">
						{#if device.last_seen}
							<span
								class={cn("size-1.5 rounded-full", marker ? MARKER_DOT[marker] : "")}
								aria-hidden="true"
							></span>
							<RelativeTime iso={device.last_seen} />
						{:else}
							<Tooltip label={NO_LAST_SEEN_TITLE} tabbable>
								<span class="text-fg-muted">{NO_LAST_SEEN_LABEL}</span>
							</Tooltip>
						{/if}
					</span>
				{/if}
			</div>
			<!-- Действие - под правой подписью. Отдельной строкой во всю ширину
			     кнопка отрывалась от текста, а в шапке вставала над ним. Подписи
			     «выведено из эксплуатации» здесь нет - её уже несёт статусный
			     чип выше. -->
			<div class="flex items-center justify-between gap-2 text-dense text-fg-secondary">
				<MonoId id={device.id} copyKey={`device-${device.id}`} />
				{#if device.status === "active"}
					<Tooltip label="Вывести из эксплуатации">
						<span class="inline-flex">
							<IconButton
								glyph={PowerIcon}
								ariaLabel="Вывести из эксплуатации"
								size="sm"
								variant="danger"
								onclick={() => onDecommission(device)}
							/>
						</span>
					</Tooltip>
				{/if}
			</div>
		</li>
	{:else}
		<li class="text-fg-muted">{emptyLabel}</li>
	{/each}
</ol>

<div class="hidden min-w-0 md:block">
	<DataTable
		tableId="devices"
		{columns}
		{rows}
		{loading}
		{skeletonRows}
		{error}
		{onRetry}
		rowKey={(row) => row.id}
		caption="Устройства: идентификатор, модель, версия, статус и последняя отметка"
	>
		{#snippet emptySnippet()}
			<span class="text-fg-muted">{emptyLabel}</span>
		{/snippet}
		{#snippet cell(id, device)}
			{@const marker = markerOf(device)}
			{#if id === "device"}
				<MonoId id={device.id} copyKey={`device-${device.id}`} />
			{:else if id === "model"}
				<button
					type="button"
					title="Отфильтровать по модели"
					onclick={() => onModelFilter(device.device_model)}
					class="text-left font-medium text-fg-primary underline-offset-4 hover:text-accent-text hover:underline"
				>
					{device.device_model}
				</button>
			{:else if id === "version"}
				<span class="flex items-center gap-2">
					<span class="font-mono tabular-nums text-fg-primary">{device.current_version}</span>
					{#if isOnTargetVersion(device.device_model, device.current_version, runningTargetVersionByModel)}
						<Tooltip label={TARGET_CHIP_TITLE}>
							<span
								class="inline-flex h-5 items-center gap-1 rounded-chip border border-border-subtle px-1.5 text-micro text-fg-secondary"
							>
								<span class="size-1.5 rounded-full bg-state-success" aria-hidden="true"></span>
								целевая
							</span>
						</Tooltip>
					{/if}
				</span>
			{:else if id === "status"}
				<StatusChip meta={DEVICE_STATUS[device.status]} raw={device.status} size="sm" />
			{:else if id === "lastSeen"}
				{#if device.last_seen}
					<span class="flex items-center justify-end gap-1.5">
						<Tooltip label={MARKER_TITLE} tabbable>
							<span class="inline-flex items-center gap-1.5">
								<span
									class={cn("size-1.5 rounded-full", marker ? MARKER_DOT[marker] : "")}
									aria-hidden="true"
								></span>
								<span class="sr-only">{MARKER_TITLE}</span>
							</span>
						</Tooltip>
						<RelativeTime iso={device.last_seen} />
					</span>
				{:else}
					<Tooltip label={NO_LAST_SEEN_TITLE} tabbable>
						<span class="text-fg-muted">{NO_LAST_SEEN_LABEL}</span>
					</Tooltip>
				{/if}
			{:else if id === "created"}
				<RelativeTime iso={device.created_at} />
			{:else if id === "actions"}
				{#if device.status === "active"}
					<Tooltip label="Вывести из эксплуатации">
						<span class="inline-flex">
							<IconButton
								glyph={PowerIcon}
								ariaLabel="Вывести из эксплуатации"
								size="sm"
								variant="danger"
								onclick={() => onDecommission(device)}
							/>
						</span>
					</Tooltip>
				{:else}
					<span class="text-dense text-fg-muted">Выведено</span>
				{/if}
			{/if}
		{/snippet}
	</DataTable>
</div>
