<script lang="ts">
	import type { Snippet } from "svelte";
	import { Dialog } from "bits-ui";

	/** Левый slide-over для мобильной навигации (240ms, §8.6). */
	let {
		open,
		onOpenChange,
		label,
		children,
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		label: string;
		children: Snippet;
	} = $props();
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Portal>
		<Dialog.Overlay class="overlay-in fixed inset-0 z-40 bg-black/55" />
		<Dialog.Content
			aria-label={label}
			class="drawer-in fixed inset-y-0 left-0 z-50 flex w-60 flex-col border-r border-border-subtle bg-bg-surface outline-none"
		>
			<Dialog.Title class="sr-only">{label}</Dialog.Title>
			{@render children()}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
