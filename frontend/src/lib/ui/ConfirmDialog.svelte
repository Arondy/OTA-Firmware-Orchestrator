<script lang="ts">
	import type { Snippet } from "svelte";
	import Modal from "./Modal.svelte";
	import Button from "./Button.svelte";

	/**
	 * Подтверждение опасного действия. Пока идёт запрос (`busy`), диалог не
	 * закрывается и обе кнопки выключены: повторная отправка невозможна.
	 */
	let {
		open,
		onOpenChange,
		title,
		body,
		confirmLabel,
		tone = "default",
		busy = false,
		details,
		onconfirm,
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		title: string;
		body?: string;
		confirmLabel: string;
		tone?: "default" | "danger";
		busy?: boolean;
		details?: Snippet;
		onconfirm: () => void;
	} = $props();

	function requestOpen(next: boolean): void {
		if (!next && busy) return;
		onOpenChange(next);
	}
</script>

<Modal {open} onOpenChange={requestOpen} {title} closeOnOutsideClick={!busy}>
	{#if body}
		<p class="text-table text-fg-secondary">{body}</p>
	{/if}
	{#if details}
		<div class="mt-2">{@render details()}</div>
	{/if}

	{#snippet footer()}
		<Button variant="ghost" disabled={busy} onclick={() => requestOpen(false)}>Отмена</Button>
		<Button variant={tone === "danger" ? "danger" : "primary"} loading={busy} onclick={onconfirm}>
			{confirmLabel}
		</Button>
	{/snippet}
</Modal>
