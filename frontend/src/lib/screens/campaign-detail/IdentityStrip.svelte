<script lang="ts">
	import type { Campaign, FirmwareVersion } from "$lib/api/types";
	import { middleTruncate, shortId } from "$lib/format/id";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import ExternalLink from "$lib/ui/ExternalLink.svelte";
	import MonoId from "$lib/ui/MonoId.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";

	/**
	 * Полоса сведений о кампании: одна строка, разделённая волосяными
	 * линиями, без карточек. Подписи 11px `fg-muted`, значения 13px.
	 */
	let { campaign, firmware }: { campaign: Campaign; firmware: FirmwareVersion | undefined } =
		$props();
</script>

<div class="grid gap-x-6 gap-y-3 border-y border-border-subtle py-3 sm:grid-cols-2 lg:grid-cols-4">
	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Модель</span>
		<span class="text-table text-fg-primary">{campaign.device_model}</span>
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Версия</span>
		{#if firmware}
			<span class="font-mono text-table tabular-nums text-fg-primary">{firmware.fw_version}</span>
		{:else}
			<span class="text-table text-fg-muted">нет в реестре</span>
		{/if}
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Кампания</span>
		<MonoId id={campaign.id} copyKey="campaign-detail-id" />
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Прошивка</span>
		{#if firmware}
			<span class="flex items-center gap-1.5">
				<CopyButton
					text={firmware.id}
					copyKey="campaign-detail-firmware"
					display={shortId(firmware.id)}
				/>
				<a
					href="/firmware?model={encodeURIComponent(campaign.device_model)}"
					class="text-table text-fg-secondary underline-offset-4 hover:text-accent-text hover:underline"
				>
					в реестре
				</a>
			</span>
		{:else}
			<span class="text-table text-fg-muted">нет</span>
		{/if}
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Создана</span>
		<RelativeTime iso={campaign.created_at} />
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Запущена</span>
		{#if campaign.started_at}
			<RelativeTime iso={campaign.started_at} />
		{:else}
			<span class="text-table text-fg-muted">нет</span>
		{/if}
	</div>

	<div class="grid content-start gap-0.5">
		<span class="text-micro text-fg-muted">Завершена</span>
		{#if campaign.completed_at}
			<RelativeTime iso={campaign.completed_at} />
		{:else}
			<span class="text-table text-fg-muted">нет</span>
		{/if}
	</div>

	{#if firmware}
		<div class="grid content-start gap-0.5 sm:col-span-2 lg:col-span-1">
			<span class="text-micro text-fg-muted">Бинарник и контрольная сумма</span>
			<span class="flex flex-wrap items-center gap-x-3 gap-y-1">
				<Tooltip label={firmware.binary_url}>
					<span class="inline-flex max-w-56">
						<ExternalLink href={firmware.binary_url} class="truncate text-table">
							{middleTruncate(firmware.binary_url)}
						</ExternalLink>
					</span>
				</Tooltip>
				<Tooltip label={firmware.fw_checksum}>
					<span class="inline-flex">
						<CopyButton
							text={firmware.fw_checksum}
							copyKey="campaign-detail-checksum"
							display={`${firmware.fw_checksum.slice(0, 8)}…${firmware.fw_checksum.slice(-8)}`}
						/>
					</span>
				</Tooltip>
			</span>
		</div>
	{/if}
</div>
