<script lang="ts">
	import ArrowDownIcon from "phosphor-svelte/lib/ArrowDownIcon";
	import ArrowUpIcon from "phosphor-svelte/lib/ArrowUpIcon";
	import CopyIcon from "phosphor-svelte/lib/CopyIcon";
	import PlusIcon from "phosphor-svelte/lib/PlusIcon";
	import XIcon from "phosphor-svelte/lib/XIcon";
	import { campaignDraft } from "$lib/state/campaign-draft.svelte";
	import {
		STAGE_ROW_MAX,
		STAGE_ROW_MIN,
		validateStageRows,
		type StageField,
	} from "$lib/domain/stage-draft";
	import { int } from "$lib/format/number";
	import Button from "$lib/ui/Button.svelte";
	import Field from "$lib/ui/Field.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
	import NumberInput from "$lib/ui/NumberInput.svelte";

	/**
	 * Шаг 2: повторяющийся редактор стадий, 1..20 строк. `order_index`
	 * оператор не видит: номер задаётся позицией строки и пересчитывается
	 * при удалении и перестановке автоматически. Жёсткие ошибки зеркалят
	 * схему запроса; убывание охвата - предупреждение, сервер его принимает.
	 */
	let {
		attempted,
		stagesBanner,
		warnings,
		onApplyCanary,
		onApplySingle,
	}: {
		/** «Далее» уже нажимали: ошибки строк показаны. */
		attempted: boolean;
		/** Вложенная ошибка стадий с сервера (400 fields.rollout_stages). */
		stagesBanner: string | undefined;
		warnings: string[];
		onApplyCanary: () => void;
		onApplySingle: () => void;
	} = $props();

	const rows = $derived(campaignDraft.rows);
	const errors = $derived(validateStageRows(rows));
	const atMax = $derived(rows.length >= STAGE_ROW_MAX);
	const atMin = $derived(rows.length <= STAGE_ROW_MIN);

	function errorOf(index: number, field: StageField): string | undefined {
		if (!attempted) return undefined;
		return errors[index][field];
	}
</script>

<div class="grid gap-4" aria-live="polite">
	<div class="flex flex-wrap items-center gap-2">
		<Button variant="secondary" size="sm" onclick={onApplyCanary}>
			Канареечная схема: 10, 50, 100
		</Button>
		<Button variant="ghost" size="sm" onclick={onApplySingle}>Одна стадия 100%</Button>
	</div>

	{#if stagesBanner}
		<InlineBanner tone="danger" boxed>{stagesBanner}</InlineBanner>
	{/if}

	{#each warnings as warning (warning)}
		<InlineBanner tone="warning" boxed>{warning}</InlineBanner>
	{/each}

	<ol class="grid gap-3">
		{#each rows as row, index (row.key)}
			<li class="grid gap-3 rounded-panel border border-border-subtle bg-bg-surface p-3">
				<div class="flex items-center justify-between gap-2">
					<span class="text-dense font-medium text-fg-primary">Стадия {index + 1}</span>
					<span class="flex items-center gap-1">
						<IconButton
							glyph={ArrowUpIcon}
							ariaLabel="Переместить стадию вверх"
							size="sm"
							disabled={index === 0}
							onclick={() => campaignDraft.moveStage(index, -1)}
						/>
						<IconButton
							glyph={ArrowDownIcon}
							ariaLabel="Переместить стадию вниз"
							size="sm"
							disabled={index === rows.length - 1}
							onclick={() => campaignDraft.moveStage(index, 1)}
						/>
						<IconButton
							glyph={CopyIcon}
							ariaLabel="Дублировать стадию"
							size="sm"
							disabled={atMax}
							onclick={() => campaignDraft.duplicateStage(index)}
						/>
						<IconButton
							glyph={XIcon}
							ariaLabel="Удалить стадию"
							size="sm"
							disabled={atMin}
							onclick={() => campaignDraft.removeStage(index)}
						/>
					</span>
				</div>

				<div class="grid gap-3 sm:grid-cols-3">
					<Field
						label="Охват"
						for="stage-{index}-target"
						error={errorOf(index, "targetPercent")}
						required
					>
						<NumberInput
							id="stage-{index}-target"
							bind:value={row.targetPercent}
							min={1}
							max={100}
							step={1}
						>
							{#snippet unit()}%{/snippet}
						</NumberInput>
					</Field>

					<Field
						label="Мин. выборка"
						for="stage-{index}-sample"
						error={errorOf(index, "minSampleSize")}
						required
					>
						<NumberInput id="stage-{index}-sample" bind:value={row.minSampleSize} min={1} step={1}>
							{#snippet unit()}наблюд.{/snippet}
						</NumberInput>
					</Field>

					<Field
						label="Порог успеха"
						for="stage-{index}-threshold"
						error={errorOf(index, "thresholdPercent")}
						required
					>
						<NumberInput
							id="stage-{index}-threshold"
							bind:value={row.thresholdPercent}
							min={0.5}
							max={100}
							step={0.5}
						>
							{#snippet unit()}%{/snippet}
						</NumberInput>
					</Field>
				</div>
			</li>
		{/each}
	</ol>

	<div class="flex flex-wrap items-center gap-2">
		<Button
			variant="secondary"
			size="sm"
			icon={PlusIcon}
			disabled={atMax}
			onclick={() => campaignDraft.addStage()}
		>
			Добавить стадию
		</Button>
		{#if atMax}
			<span class="text-dense text-fg-muted">максимум {int(STAGE_ROW_MAX)} стадий</span>
		{/if}
		{#if atMin}
			<span class="text-dense text-fg-muted">минимум одна стадия</span>
		{/if}
	</div>
</div>
