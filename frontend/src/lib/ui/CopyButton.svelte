<script lang="ts">
	import CopyIcon from "phosphor-svelte/lib/CopyIcon";
	import CheckIcon from "phosphor-svelte/lib/CheckIcon";
	import { clipboard, COPIED_LABEL } from "$lib/state/clipboard.svelte";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	/** Моноширинный текст плюс копирование с подтверждением на 1,5 с. */
	let {
		text,
		copyKey,
		display,
		class: className,
	}: {
		text: string;
		copyKey: string;
		/** Что показать вместо полного текста (например, сокращённый UUID). */
		display?: string;
		class?: string;
	} = $props();

	const copied = $derived(clipboard.isCopied(copyKey));
</script>

<span class={cn("inline-flex min-w-0 max-w-full items-center gap-1", className)}>
	<span class="truncate font-mono text-table tabular-nums text-fg-secondary" title={text}>
		{display ?? text}
	</span>
	<button
		type="button"
		aria-label={copied ? COPIED_LABEL : `Скопировать ${display ?? text}`}
		onclick={() => clipboard.copy(copyKey, text)}
		class="shrink-0 rounded-chip p-0.5 text-fg-muted transition hover:bg-bg-raised hover:text-fg-primary"
	>
		<Icon
			glyph={copied ? CheckIcon : CopyIcon}
			size={16}
			class={copied ? "text-state-success" : ""}
		/>
	</button>
</span>
