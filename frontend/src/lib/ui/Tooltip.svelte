<script lang="ts">
	import type { Snippet } from "svelte";
	import { Tooltip } from "bits-ui";

	/**
	 * Подсказка с задержкой 1 с; никогда не единственная аффорданс действия.
	 * Задержка длинная намеренно: строки таблиц насыщены иконками, и мгновенная
	 * подсказка при простом проносе курсора перекрывает соседние данные.
	 * Триггер рендерится span-ом через child-сниппет bits-ui: кнопка вокруг
	 * кнопки - это nested-interactive (axe critical), а фокус и так живёт на
	 * внутреннем контроле. Для чисто текстовых триггеров `tabbable` добавляет
	 * tabindex, чтобы пояснение доходило до клавиатуры.
	 */
	let {
		label,
		side = "top",
		tabbable = false,
		children,
	}: {
		label: string;
		side?: "top" | "bottom" | "left" | "right";
		tabbable?: boolean;
		children: Snippet;
	} = $props();
</script>

<Tooltip.Provider delayDuration={1000}>
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props: triggerProps })}
				{#if tabbable}
					<!-- Текстовый триггер: настоящая кнопка даёт клавиатурный фокус
					     и имя для скринридера без nested-interactive. -->
					<button
						type="button"
						{...triggerProps}
						class="max-w-full cursor-default justify-self-start text-left outline-none focus-visible:outline-2 focus-visible:outline-[var(--ring)]"
					>
						{@render children()}
					</button>
				{:else}
					<span {...triggerProps} class="max-w-full justify-self-start text-left outline-none">
						{@render children()}
					</span>
				{/if}
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Portal>
			<Tooltip.Content
				{side}
				sideOffset={6}
				class="z-50 max-w-64 rounded-chip border border-border-subtle bg-bg-raised px-2 py-1 text-dense text-fg-secondary shadow-pop"
			>
				{label}
			</Tooltip.Content>
		</Tooltip.Portal>
	</Tooltip.Root>
</Tooltip.Provider>
