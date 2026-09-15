<script lang="ts">
	import type { FirmwareVersion } from "$lib/api/types";
	import type { CampaignRow } from "$lib/domain/campaign-row";
	import { topModels } from "$lib/domain/campaign-row";
	import { int } from "$lib/format/number";
	import Panel from "$lib/ui/Panel.svelte";

	let {
		rows,
		firmware,
		firmwareCount,
	}: {
		rows: CampaignRow[];
		firmware: FirmwareVersion[];
		firmwareCount: number;
	} = $props();

	const latestFirmware = $derived(
		[...firmware].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at)).slice(0, 3),
	);

	const models = $derived(topModels(rows, 5));

	const modelCount = $derived(new Set(rows.map((row) => row.model)).size);
</script>

<div class="grid gap-4">
	<Panel title="Прошивки">
		{#snippet actions()}
			<span class="tabular-nums text-table text-fg-secondary">{int(firmwareCount)}</span>
		{/snippet}
		{#if firmwareCount === 0}
			<p class="text-table text-fg-muted">прошивок пока нет</p>
		{:else}
			<ul class="grid gap-1">
				{#each latestFirmware as item (item.id)}
					<li class="flex items-baseline gap-2 text-table">
						<span class="min-w-0 truncate text-fg-primary">{item.device_model}</span>
						<span class="ml-auto font-mono text-fg-secondary">{item.fw_version}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</Panel>

	<Panel title="Модели">
		{#snippet actions()}
			<span class="tabular-nums text-table text-fg-secondary">{int(modelCount)}</span>
		{/snippet}
		{#if models.length === 0}
			<p class="text-table text-fg-muted">моделей пока нет</p>
		{:else}
			<ul class="grid gap-1">
				{#each models as item (item.model)}
					<li class="flex items-baseline gap-2 text-table">
						<a
							href="/devices?model={encodeURIComponent(item.model)}"
							class="min-w-0 truncate text-fg-primary hover:text-accent-text"
						>
							{item.model}
						</a>
						<span class="ml-auto tabular-nums text-fg-secondary">
							{int(item.count)}
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	</Panel>
</div>
