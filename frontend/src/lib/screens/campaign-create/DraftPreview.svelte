<script lang="ts">
	import type { Stage } from "$lib/api/types";
	import { campaignDraft } from "$lib/state/campaign-draft.svelte";
	import { percentToFraction } from "$lib/domain/stage-draft";
	import { int, plural } from "$lib/format/number";
	import Panel from "$lib/ui/Panel.svelte";
	import StagePipeline from "$lib/viz/StagePipeline.svelte";

	/**
	 * Живой предпросмотр собираемой воронки: единственная «сводка» мастера.
	 * Стадии для рисунка синтезируются из черновика: рисунок ждет полный
	 * тип стадии, но содержательные поля здесь настоящие, из формы.
	 */
	const previewRows = $derived(campaignDraft.rows.filter((row) => row.targetPercent !== undefined));

	const previewStages = $derived.by<Stage[]>(() =>
		previewRows.map((row, index) => ({
			id: `preview-${row.key}`,
			campaign_id: "preview",
			order_index: index,
			target_percent: row.targetPercent ?? 0,
			min_sample_size: row.minSampleSize ?? 0,
			success_threshold: percentToFraction(row.thresholdPercent ?? 0),
			status: "pending",
		})),
	);

	const summary = $derived.by(() => {
		if (previewRows.length === 0) return "Стадии ещё не заданы";
		return (
			`${previewRows.length} ${plural(previewRows.length, ["стадия", "стадии", "стадий"])}: ` +
			previewRows
				.map(
					(row, index) =>
						`${index + 1}: ${int(row.targetPercent ?? 0)}% (мин. ${int(row.minSampleSize ?? 0)} ${plural(row.minSampleSize ?? 0, ["наблюдение", "наблюдения", "наблюдений"])}, порог ${int(row.thresholdPercent ?? 0)}%)`,
				)
				.join(", ")
		);
	});
</script>

<Panel title="Предпросмотр воронки">
	<div class="grid gap-3">
		{#if previewStages.length > 0}
			<StagePipeline stages={previewStages} mode="compact" />
		{:else}
			<p class="text-table text-fg-muted">Заполните охват стадий на шаге 2</p>
		{/if}
		<p class="text-dense text-fg-secondary">{summary}</p>
	</div>
</Panel>
