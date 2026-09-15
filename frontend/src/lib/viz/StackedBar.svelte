<script lang="ts">
	import type { CampaignStatus } from "$lib/api/types";
	import { CAMPAIGN_STATUS, TONE_CSS_VAR } from "$lib/domain/status";
	import { int } from "$lib/format/number";
	import { cn } from "$lib/ui/cn";

	/**
	 * Распределение кампаний по статусам: одна полоса, подписи несут и статус,
	 * и количество, отдельная легенда не дублируется (00-CONTEXT §8.5).
	 */
	let {
		counts,
		class: className,
	}: {
		counts: Record<CampaignStatus, number>;
		class?: string;
	} = $props();

	const STATUSES = Object.keys(CAMPAIGN_STATUS) as CampaignStatus[];
	const total = $derived(STATUSES.reduce((sum, status) => sum + counts[status], 0));

	const segments = $derived.by(() => {
		let cursor = 0;
		return STATUSES.filter((status) => counts[status] > 0).map((status) => {
			const width = total === 0 ? 0 : (counts[status] / total) * 100;
			const x = cursor;
			cursor += width;
			return { status, x, width };
		});
	});
</script>

<div class={cn("grid gap-2", className)}>
	<svg
		viewBox="0 0 100 6"
		preserveAspectRatio="none"
		role="img"
		aria-label={`Кампаний по статусам: ${STATUSES.map((status) => `${CAMPAIGN_STATUS[status].label} - ${counts[status]}`).join(", ")}`}
		class="h-2.5 w-full"
	>
		{#each segments as segment (segment.status)}
			<rect
				x={segment.x}
				y="0"
				width={segment.width}
				height="6"
				fill={TONE_CSS_VAR[CAMPAIGN_STATUS[segment.status].tone]}
			>
				<title>{CAMPAIGN_STATUS[segment.status].label}: {int(counts[segment.status])}</title>
			</rect>
		{/each}
	</svg>
</div>
