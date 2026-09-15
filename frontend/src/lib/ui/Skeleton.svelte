<script lang="ts">
	import { cn } from "./cn";

	/** Заготовка под будущий контент: форма совпадает с формой данных. */
	let {
		shape = "text",
		rows = 3,
		class: className,
	}: {
		shape?: "text" | "row" | "chip" | "metric" | "table";
		rows?: number;
		class?: string;
	} = $props();

	const WIDTHS = ["w-full", "w-5/6", "w-2/3", "w-3/4", "w-1/2"];
</script>

{#if shape === "chip"}
	<span class={cn("skeleton-pulse inline-block h-6 w-24 rounded-chip bg-bg-inset", className)}
	></span>
{:else if shape === "metric"}
	<div class={cn("grid gap-2", className)}>
		<span class="skeleton-pulse block h-3 w-20 rounded-chip bg-bg-inset"></span>
		<span class="skeleton-pulse block h-8 w-28 rounded-chip bg-bg-inset"></span>
	</div>
{:else if shape === "row"}
	<div class={cn("flex h-10 items-center gap-3", className)}>
		<span class="skeleton-pulse block size-4 rounded-chip bg-bg-inset"></span>
		<span class="skeleton-pulse block h-3 flex-1 rounded-chip bg-bg-inset"></span>
		<span class="skeleton-pulse block h-3 w-16 rounded-chip bg-bg-inset"></span>
	</div>
{:else if shape === "table"}
	<div class={cn("grid gap-1", className)} role="status" aria-label="Загрузка таблицы">
		{#each Array.from({ length: rows }) as _, index (index)}
			<div class="flex h-10 items-center gap-3 border-b border-border-subtle">
				<span class="skeleton-pulse block size-4 rounded-chip bg-bg-inset"></span>
				<span class="skeleton-pulse block h-3 flex-1 rounded-chip bg-bg-inset"></span>
				<span class="skeleton-pulse block h-3 w-20 rounded-chip bg-bg-inset"></span>
				<span class="skeleton-pulse block h-3 w-12 rounded-chip bg-bg-inset"></span>
			</div>
		{/each}
	</div>
{:else}
	<div class={cn("grid gap-2", className)}>
		{#each Array.from({ length: rows }) as _, index (index)}
			<span
				class={cn(
					"skeleton-pulse block h-3 rounded-chip bg-bg-inset",
					WIDTHS[index % WIDTHS.length],
				)}
			></span>
		{/each}
	</div>
{/if}
