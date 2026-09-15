<script lang="ts">
	import MagnifyingGlassIcon from "phosphor-svelte/lib/MagnifyingGlassIcon";
	import Icon from "./Icon.svelte";
	import Kbd from "./Kbd.svelte";
	import { getFieldContext } from "./field-context";
	import { inputClasses } from "./input-classes";
	import { cn } from "./cn";

	/**
	 * Единое поле поиска на все экраны: иконка лупы справа, за ней подсказка
	 * хоткея «/» (скрывается, пока поле в фокусе), атрибут `data-global-search`
	 * для глобального обработчика клавиатуры.
	 *
	 * Необязательные `suggestions` дают тот же popup-выбор, что у SearchSelect:
	 * при фокусе под полем раскрывается отфильтрованный список подсказок
	 * (например, названия моделей), стрелки и Enter работают, Esc закрывает
	 * список или очищает поле. Свободный текст остаётся валидным значением:
	 * поиск не обязан выбирать вариант из списка.
	 */
	let {
		id,
		value = $bindable(""),
		placeholder,
		suggestions = [],
		ariaLabel,
		class: className,
	}: {
		id: string;
		value?: string;
		placeholder?: string;
		/** Значения для popup-подсказок под полем. */
		suggestions?: string[];
		ariaLabel?: string;
		class?: string;
	} = $props();

	const field = getFieldContext();
	const invalid = $derived(field?.invalid ?? false);
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);

	let focused = $state(false);
	let open = $state(false);
	let activeIndex = $state(0);

	const matches = $derived.by(() => {
		const needle = value.trim().toLowerCase();
		return suggestions.filter(
			(item) => item.toLowerCase() !== needle && item.toLowerCase().includes(needle),
		);
	});
	const showList = $derived(open && focused && matches.length > 0);

	function choose(item: string): void {
		value = item;
		open = false;
		activeIndex = 0;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape") {
			if (showList) {
				event.stopPropagation();
				open = false;
				return;
			}
			if (value !== "") {
				event.stopPropagation();
				value = "";
			}
			return;
		}
		if (!showList) {
			if (event.key === "ArrowDown" && matches.length > 0) {
				event.preventDefault();
				open = true;
			}
			return;
		}
		if (event.key === "ArrowDown") {
			event.preventDefault();
			activeIndex = Math.min(matches.length - 1, activeIndex + 1);
		} else if (event.key === "ArrowUp") {
			event.preventDefault();
			activeIndex = Math.max(0, activeIndex - 1);
		} else if (event.key === "Enter") {
			const item = matches[activeIndex];
			if (item) {
				event.preventDefault();
				choose(item);
			}
		}
	}
</script>

<div class={cn("relative", className)}>
	<input
		{id}
		type="text"
		{placeholder}
		aria-label={ariaLabel}
		aria-invalid={invalid || undefined}
		aria-describedby={describedBy}
		role={suggestions.length > 0 ? "combobox" : undefined}
		aria-controls={suggestions.length > 0 ? `${id}-suggestions` : undefined}
		aria-expanded={suggestions.length > 0 ? showList : undefined}
		aria-activedescendant={showList && matches[activeIndex]
			? `${id}-suggestion-${activeIndex}`
			: undefined}
		autocomplete="off"
		data-global-search
		bind:value
		onfocus={() => {
			focused = true;
			activeIndex = 0;
			if (suggestions.length > 0) open = true;
		}}
		onblur={() => {
			focused = false;
			open = false;
		}}
		oninput={() => {
			activeIndex = 0;
			if (suggestions.length > 0) open = true;
		}}
		onkeydown={handleKeydown}
		class={inputClasses(invalid, "pr-16")}
	/>
	<span
		class="pointer-events-none absolute top-1/2 right-2 flex -translate-y-1/2 items-center gap-1.5 text-fg-muted"
	>
		<Icon glyph={MagnifyingGlassIcon} size={16} />
		{#if !focused}
			<Kbd>/</Kbd>
		{/if}
	</span>

	{#if showList}
		<div
			id="{id}-suggestions"
			role="listbox"
			aria-label={ariaLabel ?? placeholder}
			class="absolute top-full right-0 left-0 z-40 mt-1 max-h-64 overflow-y-auto rounded-panel border border-border-subtle bg-bg-raised p-1 shadow-pop"
		>
			{#each matches as item, index (item)}
				<!-- mousedown с preventDefault: фокус остаётся в поле, иначе
				     blur закрыл бы список до клика. -->
				<button
					type="button"
					role="option"
					id="{id}-suggestion-{index}"
					aria-selected={index === activeIndex}
					onmousedown={(event) => event.preventDefault()}
					onclick={() => choose(item)}
					onmouseenter={() => (activeIndex = index)}
					class={cn(
						"flex w-full cursor-pointer items-center rounded-chip px-3 py-1.5 text-left text-ui text-fg-secondary outline-none",
						index === activeIndex && "bg-bg-inset text-fg-primary",
					)}
				>
					<span class="truncate">{item}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>
