<script lang="ts">
	import type { Campaign } from "$lib/api/types";
	import { buildTimeline } from "$lib/domain/campaign-history";
	import { stageStats } from "$lib/state/stage-stats.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import Timeline from "$lib/viz/Timeline.svelte";

	/**
	 * История переходов: только события, которые API действительно отдаёт.
	 * Второй строкой перехода - фактические выборка и процент успеха, если
	 * стадия наблюдалась живой в этом сеансе, иначе её требуемые показатели.
	 */
	let { campaign }: { campaign: Campaign } = $props();

	const entries = $derived(buildTimeline(campaign, stageStats.forCampaign(campaign.id)));
</script>

<Panel title="История переходов">
	<Timeline {entries} />
</Panel>
