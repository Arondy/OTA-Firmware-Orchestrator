<script lang="ts">
	import type { AttentionReason, CampaignRow } from "$lib/domain/campaign-row";
	import { attentionText } from "$lib/domain/campaign-row";
	import { CAMPAIGN_STATUS } from "$lib/domain/status";
	import Button from "$lib/ui/Button.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";

	let { items }: { items: { row: CampaignRow; reason: AttentionReason }[] } = $props();

	const VISIBLE = 8;
	let expanded = $state(false);

	const shown = $derived(expanded ? items : items.slice(0, VISIBLE));
</script>

<Panel tone="danger" title="Требуют внимания">
	{#if items.length > 0}
		<!-- Подсветка строки живёт в границах списка: overflow-x-hidden страхует
		     от длинных неразрывных строк. Высота по содержимому: без min-h,
		     с одной кампанией панель короткая; при переполнении список
		     скроллится в max-h. overflow-anchor:none - пересортировка строк
		     не должна двигать прокрутку страницы. -->
		<ul
			class="grid max-h-[415px] overflow-x-hidden overflow-y-auto [overflow-anchor:none] md:max-h-[336px]"
		>
			{#each shown as item (item.row.id + item.reason)}
				<li class="border-b border-border-subtle last:border-b-0">
					<a
						href="/campaigns/{item.row.id}"
						class="flex flex-wrap items-center gap-x-3 gap-y-1 px-2 py-2 hover:bg-bg-raised"
					>
						<StatusChip meta={CAMPAIGN_STATUS[item.row.status]} raw={item.row.status} size="sm" />
						<span class="min-w-0 flex-1 text-table text-fg-primary">
							{attentionText(item.row, item.reason)}
						</span>
					</a>
				</li>
			{/each}
		</ul>

		{#if items.length > VISIBLE}
			<Button variant="ghost" size="sm" class="mt-2" onclick={() => (expanded = !expanded)}>
				{expanded ? "свернуть" : `показать все (${items.length})`}
			</Button>
		{/if}
	{:else}
		<p class="py-2 text-table text-fg-muted">Ничего не требует внимания</p>
	{/if}
</Panel>
