<script lang="ts">
	import type { Snippet } from "svelte";
	import { cn } from "./cn";

	let { class: className, children }: { class?: string; children: Snippet } = $props();
</script>

<dl class={cn("grid gap-x-8 divide-y divide-border-subtle", className)}>
	{@render children()}
</dl>
