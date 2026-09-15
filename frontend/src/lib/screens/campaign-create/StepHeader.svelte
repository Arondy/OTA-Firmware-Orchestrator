<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon";
	import Icon from "$lib/ui/Icon.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Шапка мастера: три подписанных сегмента, не маркетинговый степпер.
	 * Текущий шаг - янтарный, пройденные - со значком проверки, клик возвращает
	 * назад или прыгает вперёд только через валидацию (решает маршрут).
	 * Шаг с ошибкой сервера помечен опасной точкой. На узком экране сегменты
	 * сворачиваются в «Шаг 2 из 3».
	 */
	let {
		current,
		completed,
		failed,
		canEnter,
		onEnter,
	}: {
		current: 1 | 2 | 3;
		/** Шаги, значения которых прошли клиентскую валидацию. */
		completed: ReadonlySet<number>;
		/** Шаги, на которых сервер вернул ошибку полей. */
		failed: ReadonlySet<number>;
		/** Разрешён ли переход на шаг: вперёд только через валидацию. */
		canEnter: (step: 1 | 2 | 3) => boolean;
		onEnter: (step: 1 | 2 | 3) => void;
	} = $props();

	const STEPS: { step: 1 | 2 | 3; label: string }[] = [
		{ step: 1, label: "Прошивка" },
		{ step: 2, label: "Стадии" },
		{ step: 3, label: "Проверка" },
	];
</script>

<nav aria-label="Шаги мастера" class="flex items-center gap-2">
	<span class="text-dense text-fg-secondary md:hidden">Шаг {current} из 3</span>

	<ol class="hidden items-center gap-1 md:flex">
		{#each STEPS as item (item.step)}
			{@const isCurrent = item.step === current}
			{@const isDone = completed.has(item.step) && !isCurrent}
			{@const isFailed = failed.has(item.step)}
			<li>
				<button
					type="button"
					aria-current={isCurrent ? "step" : undefined}
					disabled={!canEnter(item.step)}
					onclick={() => onEnter(item.step)}
					class={cn(
						"flex h-8 items-center gap-1.5 rounded-control px-2.5 text-dense transition-colors",
						isCurrent
							? "bg-accent-tint font-medium text-fg-primary"
							: "text-fg-secondary hover:bg-bg-raised hover:text-fg-primary",
						!canEnter(item.step) && "cursor-default opacity-60 hover:bg-transparent",
					)}
				>
					{#if isFailed}
						<span class="size-1.5 rounded-full bg-state-danger" aria-hidden="true"></span>
					{:else if isDone}
						<Icon glyph={CheckIcon} size={16} class="text-state-success" />
					{:else}
						<span class={cn("tabular-nums", isCurrent && "text-accent-text")}>{item.step}</span>
					{/if}
					<span class="truncate">{item.label}</span>
				</button>
			</li>
		{/each}
	</ol>
</nav>
