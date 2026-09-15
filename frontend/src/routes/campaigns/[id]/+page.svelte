<script lang="ts">
	import { page } from "$app/state";
	import CampaignDetailScreen from "$lib/screens/campaign-detail/CampaignDetailScreen.svelte";

	// Ключ пересоздаёт экран при смене id в адресе: ресурс, таймеры и
	// локальные состояния относятся к одной кампании и не переезжают.
	const id = $derived(page.params.id ?? "");
</script>

{#key id}
	<CampaignDetailScreen {id} />
{/key}
