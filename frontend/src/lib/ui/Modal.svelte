<script lang="ts">
	import type { Snippet } from "svelte";
	import { Dialog } from "bits-ui";

	/**
	 * Модальное окно поверх bits-ui Dialog: ловушка фокуса, Esc и клик по фону
	 * из коробки. Открытое состояние управляется снаружи, чтобы диалоги могли
	 * запретить закрытие (например, пока идёт запрос).
	 *
	 * Возврат фокуса: у управляемого диалога нет `Dialog.Trigger`, которому
	 * bits-ui вернул бы фокус сам, поэтому запоминаем элемент до открытия и
	 * фокусируем его после закрытия (§10.B: focus restored).
	 *
	 * Шкала z-index на всё приложение: 40 - подложки оверлеев, 50 - содержимое
	 * оверлеев и тултипы, тосты svelte-sonner живут выше собственных слоёв.
	 */
	let {
		open,
		onOpenChange,
		title,
		description,
		closeOnOutsideClick = true,
		footer,
		children,
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		title: string;
		description?: string;
		closeOnOutsideClick?: boolean;
		footer?: Snippet;
		children?: Snippet;
	} = $props();

	let restoreFocus: HTMLElement | undefined;

	$effect(() => {
		if (open) {
			const active = document.activeElement;
			restoreFocus = active instanceof HTMLElement && active !== document.body ? active : undefined;
		}
	});

	$effect(() => {
		if (!open && restoreFocus) {
			const el = restoreFocus;
			restoreFocus = undefined;
			// Строка могла перерисоваться поллингом: фокусируем только живой элемент.
			if (el.isConnected) el.focus();
		}
	});
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Portal>
		<Dialog.Overlay class="overlay-in fixed inset-0 z-40 bg-black/55" />
		<Dialog.Content
			aria-label={title}
			onInteractOutside={(event) => {
				if (!closeOnOutsideClick) event.preventDefault();
			}}
			class="dialog-in fixed top-1/2 left-1/2 z-50 w-[min(34rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-pop border border-border-subtle bg-bg-raised p-4 shadow-pop outline-none"
		>
			<Dialog.Title class="text-panel font-semibold tracking-[-0.01em] text-fg-primary">
				{title}
			</Dialog.Title>
			{#if description}
				<Dialog.Description class="mt-1 text-table text-fg-secondary">
					{description}
				</Dialog.Description>
			{/if}

			{#if children}
				<div class="mt-3">{@render children()}</div>
			{/if}

			{#if footer}
				<div class="mt-4 flex flex-wrap justify-end gap-2">{@render footer()}</div>
			{/if}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
