<script lang="ts">
	import type { Campaign, Stats } from "$lib/api/types";
	import { STAGE_STATUS } from "$lib/domain/status";
	import { stageNumber } from "$lib/domain/campaign-history";
	import { int, percent } from "$lib/format/number";
	import Gauge from "$lib/viz/Gauge.svelte";
	import SampleProgress from "$lib/viz/SampleProgress.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";

	/**
	 * Стадии кампании: одна панель, строки без вложенных карточек. Активная
	 * стадия отмечена полосой 2px и несёт свою выборку и процент успеха.
	 * На узком экране таблица становится списком: горизонтального скролла
	 * нет ни у страницы, ни у области данных.
	 */
	let { campaign, stats }: { campaign: Campaign; stats: Stats | undefined } = $props();

	const ordered = $derived(
		[...campaign.rollout_stages].sort((a, b) => a.order_index - b.order_index),
	);
</script>

<Panel title="Стадии">
	<!-- Узкий экран: список вместо таблицы. -->
	<ol class="grid gap-3 md:hidden">
		{#each ordered as stage (stage.id)}
			<li
				class={stage.status === "active"
					? "grid gap-1.5 border-l-2 border-accent pl-3"
					: "grid gap-1.5 border-l-2 border-border-subtle pl-3"}
			>
				<div class="flex flex-wrap items-center gap-2">
					<span class="text-table font-medium text-fg-primary">
						Стадия {stageNumber(stage.order_index)}, {int(stage.target_percent)}%
					</span>
					<StatusChip meta={STAGE_STATUS[stage.status]} raw={stage.status} size="sm" />
				</div>
				<span class="text-dense text-fg-secondary">
					порог {percent(stage.success_threshold)}, мин. выборка
					{int(stage.min_sample_size)},
					{#if stage.entered_at}
						в статусе <RelativeTime iso={stage.entered_at} />
					{:else}
						в статус не вошла
					{/if}
				</span>
				{#if stage.status === "active" && stats}
					<SampleProgress
						sampleSize={stats.sample_size}
						minSampleSize={stage.min_sample_size}
						class="w-40"
					/>
					<Gauge value={stats.success_rate} threshold={stage.success_threshold} size="sm" />
				{/if}
			</li>
		{:else}
			<li class="text-fg-muted">У кампании нет стадий</li>
		{/each}
	</ol>

	<!-- Активная стадия отмечена полосой 2px у левого края строки: номер стадии
		 отстоит от неё на внутренний отступ первой колонки. -->
	<table class="hidden w-full border-collapse text-table md:table">
		<caption class="sr-only">
			Стадии кампании: охват, минимальная выборка, порог успеха, статус и время входа
		</caption>
		<thead>
			<tr class="border-b border-border-subtle text-left text-micro text-fg-muted">
				<th class="py-2 pl-3 pr-3 font-medium" scope="col">#</th>
				<th class="py-2 pr-3 font-medium" scope="col">Охват</th>
				<th class="py-2 pr-3 font-medium" scope="col">Мин. выборка</th>
				<th class="py-2 pr-3 font-medium" scope="col">Порог успеха</th>
				<th class="py-2 pr-3 font-medium" scope="col">Статус</th>
				<th class="py-2 pr-3 font-medium" scope="col">Вошла в статус</th>
				<th class="py-2 font-medium" scope="col">Выборка и процент успеха</th>
			</tr>
		</thead>
		<tbody>
			{#each ordered as stage (stage.id)}
				<tr
					class={stage.status === "active"
						? "border-b border-border-subtle shadow-[inset_2px_0_0_0_var(--accent-fill)]"
						: "border-b border-border-subtle last:border-b-0"}
				>
					<td class="py-2.5 pl-3 pr-3 tabular-nums text-fg-secondary">
						{stageNumber(stage.order_index)}
					</td>
					<td class="py-2.5 pr-3 font-mono tabular-nums text-fg-primary">
						{int(stage.target_percent)}%
					</td>
					<td class="py-2.5 pr-3 tabular-nums text-fg-secondary">{int(stage.min_sample_size)}</td>
					<td class="py-2.5 pr-3 font-mono tabular-nums text-fg-secondary">
						{percent(stage.success_threshold)}
					</td>
					<td class="py-2.5 pr-3">
						<StatusChip meta={STAGE_STATUS[stage.status]} raw={stage.status} size="sm" />
					</td>
					<td class="py-2.5 pr-3">
						{#if stage.entered_at}
							<RelativeTime iso={stage.entered_at} />
						{:else}
							<span class="text-fg-muted">нет</span>
						{/if}
					</td>
					<td class="py-2.5">
						{#if stage.status === "active" && stats}
							<div class="flex items-center gap-4">
								<SampleProgress
									sampleSize={stats.sample_size}
									minSampleSize={stage.min_sample_size}
									class="w-40"
								/>
								<Gauge value={stats.success_rate} threshold={stage.success_threshold} size="sm" />
							</div>
						{/if}
					</td>
				</tr>
			{:else}
				<tr>
					<td colspan="7" class="py-3 text-fg-muted">У кампании нет стадий</td>
				</tr>
			{/each}
		</tbody>
	</table>
</Panel>
