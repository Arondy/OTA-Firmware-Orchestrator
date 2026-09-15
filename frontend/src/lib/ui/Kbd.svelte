<script lang="ts">
	import type { Snippet } from "svelte";

	let { children }: { children: Snippet } = $props();
</script>

<kbd
	class="rounded-chip border border-border-subtle bg-bg-inset px-1.5 py-0.5 font-mono text-dense text-fg-secondary"
>
	{@render children()}
</kbd>
