<script lang="ts">
	import type { Snippet } from "svelte";
	import { cn } from "./cn";

	let {
		label,
		mono = false,
		class: className,
		children,
	}: {
		label: string;
		mono?: boolean;
		class?: string;
		children: Snippet;
	} = $props();
</script>

<div class={cn("flex items-baseline justify-between gap-4 py-1", className)}>
	<dt class="shrink-0 text-dense text-fg-muted">{label}</dt>
	<dd class={cn("min-w-0 text-right text-table text-fg-primary", mono && "font-mono tabular-nums")}>
		{@render children()}
	</dd>
</div>
