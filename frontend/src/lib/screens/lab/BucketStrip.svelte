<script lang="ts">
	import { bucketHint } from "$lib/logic/checkin-verdict";
	import { cn } from "$lib/ui/cn";

	/**
	 * Полоса из 100 ячеек: охват активной стадии закрашен, бакет устройства
	 * подсвечен янтарным. Расчёт настоящий: FNV-1a по 16 байтам device_id и
	 * campaign_id, как в серверной реализации (§4).
	 */
	let {
		bucket,
		targetPercent,
	}: {
		bucket: number | undefined;
		targetPercent: number | undefined;
	} = $props();

	const cells = Array.from({ length: 100 }, (_, index) => index + 1);
</script>

<div class="grid gap-2">
	{#if bucket === undefined || targetPercent === undefined}
		<div
			class="grid w-fit grid-cols-10 gap-0.5"
			role="img"
			aria-label="Бакет не рассчитан: выберите кампанию"
		>
			{#each cells as cell (cell)}
				<span class="size-1 rounded-[1px] bg-bg-inset" aria-hidden="true"></span>
			{/each}
		</div>
		<p class="text-dense text-fg-muted">
			{targetPercent === undefined
				? "выберите кампанию"
				: "выберите или зарегистрируйте устройство"}
		</p>
	{:else}
		<div
			class="grid w-fit grid-cols-10 gap-0.5"
			role="img"
			aria-label={bucketHint(bucket, targetPercent)}
		>
			{#each cells as cell (cell)}
				<span
					class={cn(
						"size-1 rounded-[1px]",
						cell === bucket
							? "bg-accent"
							: cell <= targetPercent
								? "bg-accent-tint"
								: "bg-bg-inset",
					)}
					aria-hidden="true"
				></span>
			{/each}
		</div>
		<p class="text-table text-fg-secondary">
			{bucketHint(bucket, targetPercent)}
		</p>
	{/if}
</div>
