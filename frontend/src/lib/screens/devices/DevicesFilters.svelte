<script lang="ts">
	import FunnelIcon from "phosphor-svelte/lib/FunnelIcon";
	import XIcon from "phosphor-svelte/lib/XIcon";
	import { NO_LAST_SEEN_LABEL } from "$lib/domain/status";
	import Button from "$lib/ui/Button.svelte";
	import Icon from "$lib/ui/Icon.svelte";
	import Field from "$lib/ui/Field.svelte";
	import SearchSelect from "$lib/ui/SearchSelect.svelte";
	import SegmentedControl from "$lib/ui/SegmentedControl.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Фильтры устройств: модель и статус уходят на сервер и живут в URL,
	 * быстрые чипы считаются только по загруженной странице и честно это
	 * говорят: API не поддерживает фильтр по `last_seen` (§5.1).
	 */
	let {
		model = $bindable(""),
		status,
		models,
		noMarkCount,
		noMarkActive,
		onStatus,
		onNoMark,
		onReset,
		filtersActive,
	}: {
		model?: string;
		status: string;
		models: string[];
		/** Сколько устройств загруженной страницы без отметки. */
		noMarkCount: number;
		noMarkActive: boolean;
		filtersActive: boolean;
		onStatus: (status: string) => void;
		onNoMark: () => void;
		onReset: () => void;
	} = $props();

	// Подписи вариантов - с большой буквы: это выбор в блоке кнопок, а не
	// словарная подпись статуса (та остаётся строчной, §9).
	const STATUS_ITEMS = [
		{ value: "", label: "Все" },
		{ value: "active", label: "Активно" },
		{ value: "decommissioned", label: "Выведено" },
	];

	const modelItems = $derived(models.map((value) => ({ value, label: value })));
</script>

<!-- Одной строкой с переносом: чип «нет отметки» стоит вплотную к блоку
     статуса, а не уезжает в дальний угол сетки. -->
<div class="rounded-panel border border-border-subtle bg-bg-surface p-3">
	<div class="flex flex-wrap items-end gap-3">
		<Field label="Модель" class="w-full sm:w-64 lg:w-72">
			<SearchSelect
				id="device-model"
				bind:value={model}
				items={modelItems}
				ariaLabel="Модель устройства"
				placeholder="Все модели"
			/>
		</Field>

		<div class="grid gap-1.5">
			<span class="text-dense font-medium text-fg-secondary">Статус</span>
			<SegmentedControl
				name="device-status"
				value={status}
				items={STATUS_ITEMS}
				ariaLabel="Статус устройства"
				onchange={onStatus}
			/>
		</div>

		<button
			type="button"
			aria-pressed={noMarkActive}
			onclick={onNoMark}
			class={cn(
				"inline-flex h-10 items-center gap-1.5 rounded-control border px-2.5 text-dense transition-colors md:h-9",
				noMarkActive
					? "border-accent-line bg-accent-tint text-accent-text"
					: "border-border-subtle bg-bg-raised text-fg-secondary hover:text-fg-primary",
			)}
		>
			<Icon glyph={FunnelIcon} size={16} />
			{NO_LAST_SEEN_LABEL}
			<span class="tabular-nums text-fg-muted">{noMarkCount}</span>
		</button>

		{#if filtersActive}
			<Button variant="secondary" icon={XIcon} onclick={onReset}>Сбросить</Button>
		{/if}
	</div>
</div>
