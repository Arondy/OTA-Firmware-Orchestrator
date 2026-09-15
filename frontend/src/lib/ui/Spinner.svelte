<script lang="ts">
	import { cn } from "./cn";

	let { class: className }: { class?: string } = $props();
</script>

<!-- 0.875rem = 14px: размер фиксирован спецификацией, проп не нужен. -->
<svg viewBox="0 0 16 16" class={cn("spinner spin text-current", className)} aria-hidden="true">
	<circle
		cx="8"
		cy="8"
		r="6.5"
		fill="none"
		stroke="currentColor"
		stroke-opacity="0.25"
		stroke-width="2"
	/>
	<path
		d="M 8 1.5 A 6.5 6.5 0 0 1 14.5 8"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
	/>
</svg>
