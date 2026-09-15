<script lang="ts">
	import { health } from "$lib/state/health.svelte";
	import { controllerHealth } from "$lib/state/controller.svelte";
	import { absoluteDateTime } from "$lib/format/datetime";
	import { int } from "$lib/format/number";
	import LiveDot from "$lib/ui/LiveDot.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Индикатор здоровья стека в верхней полосе.
	 *
	 * Оба сервиса показываются одинаково: точка состояния, имя, а при живом
	 * сервисе - реальная задержка последнего запроса вместо слова «ОК»
	 * (точка и задержка говорят больше).
	 */
	const LABELS = {
		ok: undefined,
		down: "недоступен",
		unknown: "проверка",
	} as const;

	const orchestratorTitle = $derived(
		health.lastCheckedAt === undefined
			? "идёт первая проверка доступности"
			: `последняя проверка: ${absoluteDateTime(new Date(health.lastCheckedAt).toISOString())}`,
	);

	const controllerTitle = $derived(
		controllerHealth.lastCheckedAt === undefined
			? "идёт первая проверка доступности"
			: `последняя проверка: ${absoluteDateTime(new Date(controllerHealth.lastCheckedAt).toISOString())}`,
	);
</script>

<div aria-live="polite" class="flex items-center gap-3 text-dense">
	<Tooltip label={orchestratorTitle} tabbable>
		<span class="flex items-center gap-1.5">
			{#if health.status === "ok"}
				<LiveDot class="bg-state-success" />
			{:else}
				<span
					class={cn(
						"size-1.5 rounded-full",
						health.status === "down" ? "bg-state-danger" : "bg-state-neutral",
					)}
					aria-hidden="true"
				></span>
			{/if}
			<span class="text-fg-muted">оркестратор</span>
			{#if health.status === "ok" && health.latencyMs !== undefined}
				<span class="font-mono tabular-nums text-fg-primary">{int(health.latencyMs)} мс</span>
			{:else}
				<span
					class={cn(
						"font-medium",
						health.status === "down" ? "text-state-danger" : "text-fg-primary",
					)}
				>
					{LABELS[health.status]}
				</span>
			{/if}
		</span>
	</Tooltip>

	<span class="h-4 w-px bg-border-subtle" aria-hidden="true"></span>

	<Tooltip label={controllerTitle} tabbable>
		<span class="flex items-center gap-1.5">
			{#if controllerHealth.status === "ok"}
				<LiveDot class="bg-state-success" />
			{:else}
				<span
					class={cn(
						"size-1.5 rounded-full",
						controllerHealth.status === "down" ? "bg-state-danger" : "bg-state-neutral",
					)}
					aria-hidden="true"
				></span>
			{/if}
			<span class="text-fg-muted">контроллер</span>
			{#if controllerHealth.status === "ok" && controllerHealth.latencyMs !== undefined}
				<span class="font-mono tabular-nums text-fg-primary">
					{int(controllerHealth.latencyMs)} мс
				</span>
			{:else}
				<span
					class={cn(
						"font-medium",
						controllerHealth.status === "down" ? "text-state-danger" : "text-fg-primary",
					)}
				>
					{LABELS[controllerHealth.status]}
				</span>
			{/if}
		</span>
	</Tooltip>
</div>
