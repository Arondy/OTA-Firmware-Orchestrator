<script module lang="ts">
	/** Счётчик уникальных идентификаторов полей без `for`. */
	let fieldCounter = 0;
</script>

<script lang="ts">
	import type { Snippet } from "svelte";
	import { setFieldContext, type FieldContext } from "./field-context";
	import { cn } from "./cn";

	/**
	 * Подпись над контролом, подсказка под подписью, текст ошибки под контролом
	 * (taste-skill §4.6). Плейсхолдер подписью не является никогда.
	 *
	 * `for` обязателен только для нативных полей ввода. Кнопкам-триггерам
	 * (SearchSelect) его передавать нельзя: клик по связанному label браузер
	 * форвардит на контрол и раскрывает меню, хотя пользователь целился в текст.
	 * Такие контролы получают доступное имя собственным `aria-label`.
	 */
	let {
		label,
		for: forId,
		hint,
		error,
		required = false,
		class: className,
		children,
	}: {
		label: string;
		for?: string;
		hint?: string;
		error?: string;
		required?: boolean;
		class?: string;
		children?: Snippet;
	} = $props();

	const context: FieldContext = $state({ invalid: false });

	// Идентификаторы подсказки и ошибки нужны и без `for` (контролы-кнопки
	// получают их через aria-describedby из контекста поля).
	const uid = ++fieldCounter;
	const baseId = $derived(forId ?? `field-${uid}`);

	$effect.pre(() => {
		context.hintId = hint ? `${baseId}-hint` : undefined;
		context.errorId = error ? `${baseId}-error` : undefined;
		context.invalid = Boolean(error);
	});

	setFieldContext(context);
</script>

<div class={cn("grid gap-1.5", className)}>
	<!-- `w-fit`: подпись - grid-элемент, и по умолчанию растягивается на всю
	     ширину колонки. Вместе с `for` это делало кликабельной всю строку:
	     клик по пустому месту справа от текста (например, над полем поиска)
	     уводил фокус в поле, хотя пользователь в него не целился. Ширина по
	     содержимому оставляет кликабельной только саму подпись. -->
	<label for={forId} class="w-fit text-dense font-medium text-fg-secondary">
		{label}
		{#if required}
			<span class="text-state-danger" aria-hidden="true">*</span>
		{/if}
	</label>

	{#if hint}
		<p id={`${baseId}-hint`} class="text-dense text-fg-muted">{hint}</p>
	{/if}

	{@render children?.()}

	{#if error}
		<p id={`${baseId}-error`} class="text-dense text-state-danger">{error}</p>
	{/if}
</div>
