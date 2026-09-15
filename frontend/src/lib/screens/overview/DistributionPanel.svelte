<script lang="ts">
	import type { CampaignStatus } from "$lib/api/types";
	import type { CampaignRow } from "$lib/domain/campaign-row";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import { int } from "$lib/format/number";
	import Panel from "$lib/ui/Panel.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import StackedBar from "$lib/viz/StackedBar.svelte";

	let { rows }: { rows: CampaignRow[] } = $props();

	const ORDER: CampaignStatus[] = ["draft", "running", "paused", "completed", "rolled_back"];

	const counts = $derived.by(() => {
		const result = { draft: 0, running: 0, paused: 0, completed: 0, rolled_back: 0 };
		for (const row of rows) result[row.status] += 1;
		return result;
	});
</script>

<Panel title="Распределение кампаний">
	<StackedBar {counts} />
	<ul class="mt-3 grid gap-1">
		{#each ORDER as status (status)}
			<li class="flex items-center gap-2 text-table">
				<StatusChip meta={CAMPAIGN_STATUS[status]} raw={status} size="sm" />
				<span class="ml-auto tabular-nums text-fg-secondary">
					{int(counts[status])}
				</span>
			</li>
		{/each}
	</ul>
</Panel>
