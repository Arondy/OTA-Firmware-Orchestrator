<script lang="ts">
	import WarningIcon from "phosphor-svelte/lib/WarningIcon";
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon";
	import type { ApiError } from "$lib/api/errors";
	import Button from "./Button.svelte";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	/**
	 * Сбой всегда показывает, что случилось и что делать: перевод, подсказка,
	 * «Повторить» и раскрывающиеся технические детали для отладки (§9).
	 */
	let {
		error,
		onretry,
		class: className,
	}: {
		error: ApiError;
		onretry?: () => void;
		class?: string;
	} = $props();
</script>

<div class={cn("grid justify-items-center gap-2 px-4 py-8 text-center", className)} role="alert">
	<Icon glyph={WarningIcon} size={24} class="text-state-danger" />
	<h3 class="text-ui font-semibold text-fg-primary">{error.headline}</h3>
	{#if error.hint}
		<p class="max-w-[46ch] text-table text-fg-secondary">{error.hint}</p>
	{/if}

	{#if onretry}
		<Button variant="secondary" size="sm" class="mt-1" onclick={onretry}>Повторить</Button>
	{/if}

	<details class="mt-2 max-w-full text-left">
		<summary
			class="flex cursor-pointer items-center gap-1 text-dense text-fg-muted hover:text-fg-secondary"
		>
			<Icon glyph={CaretRightIcon} size={16} />
			Технические детали
		</summary>
		<dl class="mt-2 grid gap-1 rounded-chip bg-bg-inset p-2 font-mono text-dense text-fg-secondary">
			<div class="flex gap-2">
				<dt class="text-fg-muted">запрос</dt>
				<dd class="break-all">{error.debugLine}</dd>
			</div>
			{#if error.requestId}
				<div class="flex gap-2">
					<dt class="text-fg-muted">request-id</dt>
					<dd class="break-all">{error.requestId}</dd>
				</div>
			{/if}
			{#if error.serverMessage}
				<div class="flex gap-2">
					<dt class="text-fg-muted">ответ сервера</dt>
					<dd class="break-all">{error.serverMessage}</dd>
				</div>
			{/if}
		</dl>
	</details>
</div>
