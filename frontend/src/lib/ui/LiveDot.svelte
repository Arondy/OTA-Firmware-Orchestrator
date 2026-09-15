<script lang="ts">
	import { cn } from "./cn";

	/** Пульсирующая точка «живого» состояния; пульс гаснет при reduced motion. */
	let { class: className }: { class?: string } = $props();
</script>

<span class={cn("live-pulse size-1.5 rounded-full", className)} aria-hidden="true"></span>
