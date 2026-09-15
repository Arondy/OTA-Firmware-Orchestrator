<script lang="ts">
	import { getFieldContext } from "./field-context";
	import { inputClasses } from "./input-classes";

	let {
		id,
		value = $bindable(""),
		type = "text",
		placeholder,
		disabled = false,
		mono = false,
		list,
		maxlength,
		class: className,
		...restProps
	}: {
		id: string;
		value?: string;
		type?: "text" | "email" | "url" | "password";
		placeholder?: string;
		disabled?: boolean;
		/** Моноширинный набор для машинных значений: семвер, суммы, URL. */
		mono?: boolean;
		/** id элемента datalist с подсказками значений. */
		list?: string;
		maxlength?: number;
		class?: string;
		[key: `data-${string}`]: string | boolean | undefined;
	} = $props();

	const field = getFieldContext();
	const invalid = $derived(field?.invalid ?? false);
	const describedBy = $derived(
		[field?.hintId, field?.errorId].filter(Boolean).join(" ") || undefined,
	);
</script>

<input
	{id}
	{type}
	{placeholder}
	{disabled}
	{list}
	{maxlength}
	{...restProps}
	bind:value
	aria-invalid={invalid || undefined}
	aria-describedby={describedBy}
	class={inputClasses(invalid, mono && "font-mono", className)}
/>
