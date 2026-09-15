<script lang="ts">
	import RocketLaunchIcon from "phosphor-svelte/lib/RocketLaunchIcon";
	import type { FirmwareVersion } from "$lib/api/types";
	import { semverSortKey } from "$lib/logic/semver";
	import { middleTruncate } from "$lib/format/id";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import DataTable from "$lib/ui/DataTable.svelte";
	import ExternalLink from "$lib/ui/ExternalLink.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import Skeleton from "$lib/ui/Skeleton.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import type { Column } from "$lib/ui/table-types";

	/**
	 * Реестр прошивок: сортировка клиентская и только по загруженной странице
	 * (семвер - через ключ сортировки, даты - меткой времени), порядок по
	 * умолчанию серверный, сначала новые (§7.5).
	 */
	let {
		rows,
		latestIds,
		loading,
		skeletonRows = 25,
	}: {
		rows: FirmwareVersion[];
		skeletonRows?: number;
		/** id версий с наибольшим semver внутри модели среди загруженных. */
		latestIds: ReadonlySet<string>;
		loading: boolean;
	} = $props();

	// Ширины в px зафиксированы и растягиваются драгом разделителей:
	// раскладка не зависит от содержимого строк и активных фильтров.
	// «Действия» держат ту же ширину, что в таблице кампаний: колонка
	// одинаковая во всех списках, поэтому кнопки не «прыгают» между экранами.
	// Цвет иконки повторяет смысл действия: зелёный - «новая кампания», как
	// запуск и возобновление в таблице кампаний. Подпись всё равно обязательна:
	// цвет ускоряет узнавание, а не заменяет название.
	const columns: Column<FirmwareVersion>[] = [
		{ id: "model", header: "Модель", width: 176 },
		{
			id: "version",
			header: "Версия",
			width: 160,
			sortValue: (row) => semverSortKey(row.fw_version),
		},
		{ id: "checksum", header: "Контрольная сумма", width: 208 },
		{ id: "binary", header: "Бинарник", width: 176 },
		{
			id: "created",
			header: "Создана",
			align: "right",
			width: 144,
			sortValue: (row) => Date.parse(row.created_at),
		},
		{ id: "actions", header: "Действия", align: "right", width: 112 },
	];
</script>

<!-- Узкий экран: карточки вместо таблицы (§6). -->
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
	{#each rows as row (row.id)}
		<li class="grid gap-1.5 rounded-panel border border-border-subtle bg-bg-surface p-3">
			<!-- Правая подпись стоит на уровне названия модели, а не отдельной
			     строкой под контрольной суммой: она отвечает на вопрос «когда
			     версия появилась», и читать её рядом с названием дешевле. -->
			<div class="flex flex-wrap items-center gap-2">
				<span class="text-table font-medium text-fg-primary">{row.device_model}</span>
				<span class="font-mono text-table tabular-nums text-fg-secondary">{row.fw_version}</span>
				{#if latestIds.has(row.id)}
					<span
						class="inline-flex h-5 items-center rounded-chip border border-border-subtle px-1.5 text-micro text-fg-secondary"
					>
						последняя
					</span>
				{/if}
				<span class="ml-auto text-dense text-fg-secondary">
					создана <RelativeTime iso={row.created_at} />
				</span>
			</div>
			<!-- Та же мелкая кнопка действия, что в колонке «Действия» на десктопе,
			     но под правой подписью: отдельной строкой внизу она отрывалась от
			     текста, а в шапке карточки вставала над ним. -->
			<div class="flex items-center justify-between gap-2 text-dense text-fg-secondary">
				<CopyButton
					text={row.fw_checksum}
					copyKey={`firmware-card-${row.id}`}
					display={`${row.fw_checksum.slice(0, 8)}…${row.fw_checksum.slice(-6)}`}
				/>
				<Tooltip label="Новая кампания">
					<span class="inline-flex">
						<IconButton
							glyph={RocketLaunchIcon}
							ariaLabel="Новая кампания"
							size="sm"
							variant="success"
							href="/campaigns/new?step=2&firmware={row.id}"
						/>
					</span>
				</Tooltip>
			</div>
			<Tooltip label={row.binary_url}>
				<span class="flex min-w-0">
					<ExternalLink href={row.binary_url}>{middleTruncate(row.binary_url, 30)}</ExternalLink>
				</span>
			</Tooltip>
		</li>
	{:else}
		<li class="text-fg-muted">Версий нет</li>
	{/each}
</ol>

<div class="hidden min-w-0 md:block">
	<DataTable
		tableId="firmware"
		{columns}
		{rows}
		{loading}
		{skeletonRows}
		rowKey={(row) => row.id}
		caption="Реестр версий прошивок"
	>
		{#snippet cell(id, row)}
			{#if id === "model"}
				<span class="font-medium text-fg-primary">{row.device_model}</span>
			{:else if id === "version"}
				<span class="flex items-center gap-2">
					<span class="font-mono tabular-nums text-fg-primary">{row.fw_version}</span>
					{#if latestIds.has(row.id)}
						<span
							class="inline-flex h-5 items-center rounded-chip border border-border-subtle px-1.5 text-micro text-fg-secondary"
						>
							последняя
						</span>
					{/if}
				</span>
			{:else if id === "checksum"}
				<CopyButton
					text={row.fw_checksum}
					copyKey={`firmware-${row.id}`}
					display={`${row.fw_checksum.slice(0, 8)}…${row.fw_checksum.slice(-6)}`}
				/>
			{:else if id === "binary"}
				<Tooltip label={row.binary_url}>
					<span class="inline-flex max-w-56">
						<ExternalLink href={row.binary_url} class="truncate">
							{middleTruncate(row.binary_url, 34)}
						</ExternalLink>
					</span>
				</Tooltip>
			{:else if id === "created"}
				<RelativeTime iso={row.created_at} />
			{:else if id === "actions"}
				<Tooltip label="Новая кампания">
					<span class="inline-flex">
						<IconButton
							glyph={RocketLaunchIcon}
							ariaLabel="Новая кампания"
							size="sm"
							variant="success"
							href="/campaigns/new?step=2&firmware={row.id}"
						/>
					</span>
				</Tooltip>
			{/if}
		{/snippet}
	</DataTable>
</div>
