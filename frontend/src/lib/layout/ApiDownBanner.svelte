<script lang="ts">
	import { apiAvailability } from "$lib/state/api-availability.svelte";
	import { refreshRegistry } from "$lib/state/refresh.svelte";
	import Button from "$lib/ui/Button.svelte";
	import InlineBanner from "$lib/ui/InlineBanner.svelte";
</script>

{#if apiAvailability.down}
	<div class="banner-enter grid">
		<InlineBanner tone="danger">
			<strong class="font-semibold">API недоступен.</strong>
			Показаны последние полученные данные.
			{#snippet action()}
				<Button
					variant="secondary"
					size="sm"
					loading={refreshRegistry.busy}
					onclick={() => refreshRegistry.refreshAll()}
				>
					Повторить
				</Button>
			{/snippet}
		</InlineBanner>
	</div>
{/if}
