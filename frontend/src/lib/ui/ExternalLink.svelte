<script lang="ts">
	import type { Snippet } from "svelte";
	import ArrowUpRightIcon from "phosphor-svelte/lib/ArrowUpRightIcon";
	import Icon from "./Icon.svelte";
	import { cn } from "./cn";

	let {
		href,
		class: className,
		children,
	}: { href: string; class?: string; children: Snippet } = $props();
</script>

<a
	{href}
	target="_blank"
	rel="noopener noreferrer"
	class={cn(
		"inline-flex items-center gap-1 text-table text-accent-text underline-offset-2 hover:underline",
		className,
	)}
>
	{@render children()}
	<Icon glyph={ArrowUpRightIcon} size={16} />
</a>
