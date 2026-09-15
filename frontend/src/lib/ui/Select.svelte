<script lang="ts">
	import CaretDownIcon from "phosphor-svelte/lib/CaretDownIcon";
	import type { Component } from "svelte";
	import { Select, type SelectRootProps } from "bits-ui";
	import CheckIcon from "phosphor-svelte/lib/CheckIcon";
	import Icon from "./Icon.svelte";
	import { getFieldContext } from "./field-context";
	import { inputClasses } from "./input-classes";
	import { cn } from "./cn";

	let {
		id,
		value = $bindable(""),
		items,
		placeholder,
		ariaLabel,
		disabled = false,
		class: className,
	}: {
		id: string;
		value?: string;
		items: { value: string; label: string }[];
		placeholder?: string;
		/**
		 * Доступное имя триггера: он кнопка, поэтому `Field` не должен
		 * связывать с ним подпись через `for` - клик по подписи раскрыл бы
		 * список. Подпись остаётся визуальной, имя задаётся здесь.
		 */
		ariaLabel?: string;
		disabled?: boolean;
		class?: string;
	} = $props();

	// bits-ui описывает одиночный и множественный выбор одним union-типом пропсов,
	// который не сужается через пропсы компонента; алиас фиксирует одиночную ветку.
	type SingleSelectProps = Omit<SelectRootProps, "value" | "onValueChange" | "multiple"> & {
		value?: string;
		onValueChange?: (value: string) => void;
	};
	const SelectRoot = Select.Root as unknown as Component<SingleSelectProps>;

	function handleValueChange(next: string): void {
		value = next;
	}

	// bits-ui рисует в Select.Value сырое значение, а оператору нужна подпись
	// варианта: label ищем сами по выбранному значению.
	const selectedLabel = $derived(items.find((item) => item.value === value)?.label);

	const field = getFieldContext();
	const invalid = $derived(field?.invalid ?? false);
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<SelectRoot type="single" {value} onValueChange={handleValueChange} {disabled}>
	<Select.Trigger
		{id}
		aria-label={ariaLabel}
		aria-describedby={describedBy}
		aria-invalid={invalid || undefined}
		class={inputClasses(invalid, "flex items-center gap-2 text-left", className)}
	>
		<span class={cn("truncate", !selectedLabel && "text-fg-muted")}>
			{selectedLabel ?? placeholder ?? ""}
		</span>
		<Icon glyph={CaretDownIcon} size={16} class="ml-auto text-fg-muted" />
	</Select.Trigger>

	<Select.Portal>
		<Select.Content
			side="bottom"
			sideOffset={4}
			class="z-40 max-h-72 min-w-56 overflow-y-auto rounded-panel border border-border-subtle bg-bg-raised p-1 shadow-pop"
		>
			<Select.Viewport>
				{#each items as item (item.value)}
					<Select.Item
						value={item.value}
						label={item.label}
						class="flex cursor-pointer items-center gap-2 rounded-chip px-2 py-1.5 text-ui text-fg-secondary outline-none data-[highlighted]:bg-bg-inset data-[highlighted]:text-fg-primary"
					>
						<span class="w-4 shrink-0">
							{#if value === item.value}
								<Icon glyph={CheckIcon} size={16} class="text-accent-text" />
							{/if}
						</span>
						<span class="truncate">{item.label}</span>
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</SelectRoot>
