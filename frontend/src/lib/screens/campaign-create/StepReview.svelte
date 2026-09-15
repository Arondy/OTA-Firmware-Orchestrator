<script lang="ts">
	import { campaignDraft } from "$lib/state/campaign-draft.svelte";
	import { firmwareIndex } from "$lib/state/firmware-index.svelte";
	import { int, percent } from "$lib/format/number";
	import JsonView from "$lib/ui/JsonView.svelte";
	import KeyValue from "$lib/ui/KeyValue.svelte";
	import KeyValueGrid from "$lib/ui/KeyValueGrid.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";

	/**
	 * Шаг 3: сверка перед единственным запросом. Тело запроса показано тем же
	 * объектом, который уйдёт на сервер: доли, а не проценты порога.
	 */
	const firmware = $derived(firmwareIndex.get(campaignDraft.firmwareVersionId));
	const payload = $derived(campaignDraft.payload());
</script>

<div class="grid gap-4">
	<KeyValueGrid class="sm:grid-cols-2">
		<KeyValue label="Модель">{campaignDraft.model || "не выбрана"}</KeyValue>
		<KeyValue label="Версия" mono>{firmware?.fw_version ?? "не выбрана"}</KeyValue>
		<KeyValue label="Идентификатор прошивки">
			{#if firmware}
				<MonoId id={firmware.id} copyKey="review-firmware-id" />
			{:else}
				нет
			{/if}
		</KeyValue>
		<KeyValue label="Число стадий">{int(payload.rollout_stages.length)}</KeyValue>
	</KeyValueGrid>

	<table class="w-full border-collapse text-table">
		<caption class="sr-only">Стадии будущей кампании</caption>
		<thead>
			<tr class="border-b border-border-subtle text-left text-micro text-fg-muted">
				<th class="py-2 pr-3 font-medium" scope="col">#</th>
				<th class="py-2 pr-3 font-medium" scope="col">Охват</th>
				<th class="py-2 pr-3 font-medium" scope="col">Мин. выборка</th>
				<th class="py-2 font-medium" scope="col">Порог успеха</th>
			</tr>
		</thead>
		<tbody>
			{#each payload.rollout_stages as stage (stage.order_index)}
				<tr class="border-b border-border-subtle last:border-b-0">
					<td class="py-2 pr-3 tabular-nums text-fg-secondary">{stage.order_index + 1}</td>
					<td class="py-2 pr-3 font-mono tabular-nums text-fg-primary">
						{int(stage.target_percent)}%
					</td>
					<td class="py-2 pr-3 tabular-nums text-fg-secondary">{int(stage.min_sample_size)}</td>
					<td class="py-2 font-mono tabular-nums text-fg-secondary">
						{percent(stage.success_threshold)}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>

	<details class="grid gap-2">
		<summary
			class="cursor-pointer list-none text-table text-fg-secondary underline-offset-4 hover:text-accent-text hover:underline focus-visible:outline-2 focus-visible:outline-accent"
		>
			Тело запроса
		</summary>
		<JsonView value={payload} label="POST /api/v1/campaigns" />
	</details>
</div>
