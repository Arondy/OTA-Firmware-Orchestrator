<script lang="ts">
	import CaretDownIcon from "phosphor-svelte/lib/CaretDownIcon";
	import CheckIcon from "phosphor-svelte/lib/CheckIcon";
	import type { Snippet } from "svelte";
	import { Popover } from "bits-ui";
	import Icon from "./Icon.svelte";
	import { inputClasses } from "./input-classes";
	import { cn } from "./cn";

	/**
	 * Выбор из списка: единый элемент на все экраны (устройства, прошивки,
	 * песочница, создание кампании). Внутри раскрывающейся панели всегда есть
	 * поиск и прокручиваемый список, под списком - необязательный слот действия
	 * («Зарегистрировать прошивку»).
	 *
	 * Пустое значение показывается плейсхолдером («Все модели», «без кампании»):
	 * отдельного варианта «Все модели» в списке нет, сброс делается кнопкой
	 * «Сбросить» рядом с полем.
	 *
	 * Триггер - кнопка, поэтому `Field` не должен связывать с ним подпись через
	 * `for` (клик по подписи раскрыл бы меню); доступное имя задаёт `ariaLabel`.
	 */
	let {
		id,
		value = $bindable(""),
		items,
		placeholder,
		ariaLabel,
		disabled = false,
		footer,
		onValueChange,
		class: className,
	}: {
		id: string;
		value?: string;
		items: { value: string; label: string }[];
		placeholder?: string;
		/** Доступное имя триггера и списка: подпись Field до него не доезжает. */
		ariaLabel?: string;
		disabled?: boolean;
		footer?: Snippet;
		/** Уведомление о выборе для родителей без двустороннего связывания. */
		onValueChange?: (value: string) => void;
		class?: string;
	} = $props();

	let open = $state(false);
	let query = $state("");
	let activeIndex = $state(0);
	let searchInput: HTMLInputElement | undefined = $state(undefined);

	const selectedLabel = $derived(items.find((item) => item.value === value)?.label);
	const filtered = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!needle) return items;
		return items.filter((item) => item.label.toLowerCase().includes(needle));
	});

	$effect(() => {
		if (!open) return;
		query = "";
		activeIndex = Math.max(
			0,
			items.findIndex((item) => item.value === value),
		);
		searchInput?.focus();
	});

	function select(next: string): void {
		value = next;
		open = false;
		onValueChange?.(next);
	}

	function handleSearchKeydown(event: KeyboardEvent): void {
		if (event.key === "ArrowDown") {
			event.preventDefault();
			activeIndex = Math.min(filtered.length - 1, activeIndex + 1);
		} else if (event.key === "ArrowUp") {
			event.preventDefault();
			activeIndex = Math.max(0, activeIndex - 1);
		} else if (event.key === "Enter") {
			event.preventDefault();
			const item = filtered[activeIndex];
			if (item) select(item.value);
		}
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		{id}
		type="button"
		{disabled}
		aria-haspopup="listbox"
		aria-label={ariaLabel}
		class={inputClasses(false, "flex items-center gap-2 text-left", className)}
	>
		<span class={cn("truncate", !selectedLabel && "text-fg-muted")}>
			{selectedLabel ?? placeholder ?? ""}
		</span>
		<Icon glyph={CaretDownIcon} size={16} class="ml-auto shrink-0 text-fg-muted" />
	</Popover.Trigger>

	<Popover.Content
		side="bottom"
		sideOffset={4}
		class="z-40 min-w-64 rounded-panel border border-border-subtle bg-bg-raised p-1 shadow-pop outline-none w-[var(--bits-popover-trigger-width)]"
	>
		<div class="p-1">
			<input
				bind:this={searchInput}
				type="text"
				role="combobox"
				aria-controls="{id}-listbox"
				aria-expanded="true"
				aria-activedescendant={filtered[activeIndex] ? `${id}-opt-${activeIndex}` : undefined}
				aria-label="Поиск по списку"
				placeholder="Поиск"
				value={query}
				oninput={(event) => {
					query = event.currentTarget.value;
					activeIndex = 0;
				}}
				onkeydown={handleSearchKeydown}
				class={inputClasses(false, "w-full")}
			/>
		</div>

		<!-- Текст варианта выровнен по левой границе текста в поле поиска выше:
		     тот же отступ 12px, без колонки под отметку выбора. Отметка - справа. -->
		<div
			id="{id}-listbox"
			role="listbox"
			aria-label={ariaLabel ?? placeholder}
			class="max-h-64 overflow-y-auto"
		>
			{#each filtered as item, index (item.value)}
				<button
					type="button"
					role="option"
					id="{id}-opt-{index}"
					aria-selected={value === item.value}
					onclick={() => select(item.value)}
					onmouseenter={() => (activeIndex = index)}
					class={cn(
						"flex w-full cursor-pointer items-center gap-2 rounded-chip px-3 py-1.5 text-left text-ui text-fg-secondary outline-none",
						index === activeIndex && "bg-bg-inset text-fg-primary",
						"focus-visible:bg-bg-inset focus-visible:text-fg-primary",
					)}
				>
					<span class="truncate">{item.label}</span>
					{#if value === item.value}
						<Icon glyph={CheckIcon} size={16} class="ml-auto shrink-0 text-accent-text" />
					{/if}
				</button>
			{:else}
				<p class="px-3 py-1.5 text-dense text-fg-muted">Ничего не найдено</p>
			{/each}
		</div>

		{#if footer}
			<div class="mt-1 border-t border-border-subtle p-1">{@render footer()}</div>
		{/if}
	</Popover.Content>
</Popover.Root>
