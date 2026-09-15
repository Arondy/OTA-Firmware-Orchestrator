<script lang="ts">
	import { relativeTime, absoluteDateTimeTitle } from "$lib/format/datetime";
	import { clock } from "$lib/state/clock.svelte";

	/** Относительное время на общих часах: один тик на приложение, не на строку. */
	let { iso }: { iso: string | undefined } = $props();
</script>

<time datetime={iso} title={absoluteDateTimeTitle(iso)} class="text-table text-fg-secondary">
	{relativeTime(iso, clock.now)}
</time>
