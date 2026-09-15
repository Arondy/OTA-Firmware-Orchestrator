<script lang="ts">
	import type { Device } from "$lib/api/types";
	import { bucketOf } from "$lib/logic/bucket";
	import { isSemver, parseSemver } from "$lib/logic/semver";
	import { DEVICE_STATUS, NO_LAST_SEEN_LABEL, NO_LAST_SEEN_TITLE } from "$lib/domain/status";
	import { shortId } from "$lib/format/id";
	import { mutation } from "$lib/state/mutation.svelte";
	import Button from "$lib/ui/Button.svelte";
	import CopyButton from "$lib/ui/CopyButton.svelte";
	import Field from "$lib/ui/Field.svelte";
	import KeyValue from "$lib/ui/KeyValue.svelte";
	import KeyValueGrid from "$lib/ui/KeyValueGrid.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import RelativeTime from "$lib/ui/RelativeTime.svelte";
	import SearchSelect from "$lib/ui/SearchSelect.svelte";
	import SegmentedControl from "$lib/ui/SegmentedControl.svelte";
	import StatusChip from "$lib/ui/StatusChip.svelte";
	import TextInput from "$lib/ui/TextInput.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import BucketStrip from "./BucketStrip.svelte";

	/**
	 * Устройство песочницы. Оператор явно выбирает один из двух вариантов -
	 * существующее устройство или новое: форма регистрации и выбор не
	 * показываются одновременно и не сбивают друг друга.
	 *
	 * В списке выбора только активные устройства первой страницы реестра
	 * модели: выведенные из эксплуатации в прогоне не участвуют, а значит
	 * и статус в подписи варианта не нужен.
	 */
	let {
		devices,
		device,
		campaignId,
		campaignModel,
		targetVersion,
		targetPercent,
		registerModel = $bindable(""),
		registerVersion = $bindable(""),
		onPick,
		onRegister,
	}: {
		devices: Device[];
		device: Device | undefined;
		campaignId: string | undefined;
		campaignModel: string;
		targetVersion: string | undefined;
		targetPercent: number | undefined;
		/** Значения формы регистрации живут в маршруте: шаг 1 степпера тот же. */
		registerModel?: string;
		registerVersion?: string;
		onPick: (device: Device | undefined) => void;
		onRegister: (model: string, version: string) => Promise<Device | undefined>;
	} = $props();

	type DeviceMode = "existing" | "new";

	const MODE_ITEMS: { value: DeviceMode; label: string }[] = [
		{ value: "existing", label: "Существующее" },
		{ value: "new", label: "Новое" },
	];

	let mode = $state<DeviceMode>("existing");
	let pickedId = $state("");
	let attempted = $state(false);

	/** Версия заведомо ниже целевой: минус мажор, иначе минус минор. */
	function versionBelow(target: string | undefined): string {
		const parsed = parseSemver(target ?? "");
		if (!parsed) return "";
		if (parsed.major > 0) return `${parsed.major - 1}.0.0`;
		if (parsed.minor > 0) return `0.${parsed.minor - 1}.0`;
		return "0.0.0";
	}

	// Префиллы следуют за сменой кампании; маршрут держит значения формы.
	$effect(() => {
		registerModel = campaignModel;
		registerVersion = versionBelow(targetVersion);
		attempted = false;
	});

	// Только активные устройства: decommissioned в песочнице не используются,
	// поэтому и статус в подпись варианта не попадает.
	const deviceItems = $derived(
		devices
			.filter((row) => row.status === "active")
			.map((row) => ({
				value: row.id,
				label: `${shortId(row.id)}, ${row.current_version}`,
			})),
	);

	const bucket = $derived(device && campaignId ? bucketOf(device.id, campaignId) : undefined);

	const errors = $derived.by(() => {
		const next: Partial<Record<"model" | "version", string>> = {};
		const model = registerModel.trim();
		if (model.length < 2 || model.length > 64) next.model = "От 2 до 64 символов";
		if (!isSemver(registerVersion.trim())) next.version = "Semver, например 1.0.0";
		return next;
	});

	const registerPending = $derived(mutation.isPending("lab-register"));

	function switchMode(next: string): void {
		mode = next as DeviceMode;
		attempted = false;
	}

	async function registerHere(): Promise<void> {
		attempted = true;
		if (Object.keys(errors).length > 0) return;
		const created = await onRegister(registerModel.trim(), registerVersion.trim());
		if (created) {
			pickedId = created.id;
			mode = "existing";
		}
	}
</script>

<Panel title="Устройство">
	<div class="grid gap-4">
		<SegmentedControl
			name="lab-device-mode"
			value={mode}
			items={MODE_ITEMS}
			ariaLabel="Источник устройства"
			onchange={switchMode}
		/>

		{#if mode === "existing"}
			<SearchSelect
				id="lab-device"
				value={pickedId}
				items={deviceItems}
				ariaLabel="Существующее устройство"
				placeholder={campaignModel ? "Выберите устройство" : "Сначала выберите кампанию"}
				disabled={!campaignModel}
				onValueChange={(id) => {
					pickedId = id;
					onPick(devices.find((row) => row.id === id));
				}}
			/>
		{:else}
			<div class="grid gap-3 sm:grid-cols-2">
				<Field
					label="Модель нового устройства"
					for="lab-new-model"
					error={attempted ? errors.model : undefined}
				>
					<TextInput id="lab-new-model" bind:value={registerModel} maxlength={64} />
				</Field>
				<Field
					label="Начальная версия"
					for="lab-new-version"
					error={attempted ? errors.version : undefined}
				>
					<TextInput id="lab-new-version" bind:value={registerVersion} mono />
				</Field>
			</div>
			<div>
				<Button
					variant="secondary"
					size="sm"
					loading={registerPending}
					onclick={() => void registerHere()}
				>
					Зарегистрировать устройство
				</Button>
			</div>
		{/if}

		{#if device}
			<KeyValueGrid class="border-t border-border-subtle pt-3 sm:grid-cols-2">
				<KeyValue label="Идентификатор">
					<CopyButton text={device.id} copyKey="lab-device-id" display={shortId(device.id)} />
				</KeyValue>
				<KeyValue label="Модель">{device.device_model}</KeyValue>
				<KeyValue label="Текущая версия" mono>{device.current_version}</KeyValue>
				<KeyValue label="Статус">
					<StatusChip meta={DEVICE_STATUS[device.status]} raw={device.status} size="sm" />
				</KeyValue>
				<KeyValue label="Последняя отметка">
					{#if device.last_seen}
						<RelativeTime iso={device.last_seen} />
					{:else}
						<Tooltip label={NO_LAST_SEEN_TITLE}>
							<span class="text-fg-muted">{NO_LAST_SEEN_LABEL}</span>
						</Tooltip>
					{/if}
				</KeyValue>
			</KeyValueGrid>
		{/if}

		<div class="border-t border-border-subtle pt-3">
			<BucketStrip {bucket} {targetPercent} />
		</div>
	</div>
</Panel>
