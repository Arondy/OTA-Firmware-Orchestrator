<script lang="ts">
	import { shortId } from "$lib/format/id";
	import CopyButton from "./CopyButton.svelte";
	import Tooltip from "./Tooltip.svelte";

	/** Сокращённый UUID с копированием; полное значение в подсказке. */
	let { id, copyKey }: { id: string; copyKey: string } = $props();
</script>

<Tooltip label={id}>
	<CopyButton text={id} {copyKey} display={shortId(id)} />
</Tooltip>
