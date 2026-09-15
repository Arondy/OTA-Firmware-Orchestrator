<script lang="ts">
	import type { Snippet } from "svelte";
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import { refreshRegistry } from "$lib/state/refresh.svelte";
	import { cycleTheme } from "$lib/state/theme.svelte";
	import Drawer from "$lib/ui/Drawer.svelte";
	import ApiDownBanner from "./ApiDownBanner.svelte";
	import ShortcutsModal from "./ShortcutsModal.svelte";
	import Sidebar from "./Sidebar.svelte";
	import Topbar from "./Topbar.svelte";

	const STORAGE_KEY = "ota-sidebar-collapsed";

	let { children }: { children: Snippet } = $props();

	let collapsed = $state(localStorage.getItem(STORAGE_KEY) === "1");
	let drawerOpen = $state(false);
	let shortcutsOpen = $state(false);

	function toggleCollapsed(): void {
		collapsed = !collapsed;
		localStorage.setItem(STORAGE_KEY, collapsed ? "1" : "0");
	}

	// Навигация закрывает мобильное меню; фокус возвращает сам диалог.
	let lastPath = $state("");
	$effect(() => {
		const current = page.url.pathname;
		if (current === lastPath) return;
		lastPath = current;
		drawerOpen = false;
	});

	/**
	 * Клавиатура приложения: один слушатель на window. В полях ввода
	 * горячие клавиши не срабатывают (кроме Esc), последовательность `g`
	 * живёт 900 мс и гаснет сама.
	 */
	const GOTO_KEYS: Record<string, string> = {
		o: "/",
		c: "/campaigns",
		d: "/devices",
		f: "/firmware",
		l: "/lab",
	};

	let gPending = false;
	let gTimer: number | undefined;

	$effect(() => {
		function onKey(event: KeyboardEvent): void {
			const target = event.target as HTMLElement | null;
			const inField =
				target !== null &&
				(target.tagName === "INPUT" ||
					target.tagName === "TEXTAREA" ||
					target.tagName === "SELECT" ||
					target.isContentEditable);

			if (event.key === "Escape") {
				gPending = false;
				return;
			}
			if (inField || event.metaKey || event.ctrlKey || event.altKey) return;

			if (gPending) {
				gPending = false;
				window.clearTimeout(gTimer);
				const to = GOTO_KEYS[event.key];
				if (to) {
					event.preventDefault();
					void goto(to);
				}
				return;
			}

			if (event.key === "?") {
				event.preventDefault();
				shortcutsOpen = true;
				return;
			}
			if (event.key === "/") {
				const input = document.querySelector<HTMLInputElement>("input[data-global-search]");
				if (input) {
					event.preventDefault();
					input.focus();
				}
				return;
			}
			if (event.key === "g") {
				gPending = true;
				window.clearTimeout(gTimer);
				gTimer = window.setTimeout(() => {
					gPending = false;
				}, 900);
				return;
			}
			if (event.key === "r") {
				event.preventDefault();
				void refreshRegistry.refreshAll();
				return;
			}
			if (event.key === "t") {
				event.preventDefault();
				cycleTheme();
			}
		}

		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	});
</script>

<a
	href="#content"
	class="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-control focus:border focus:border-border-strong focus:bg-bg-raised focus:px-3 focus:py-2 focus:text-ui focus:text-fg-primary"
>
	К содержимому
</a>

<div class="grid min-h-[100dvh] grid-cols-1 grid-rows-[3.5rem_1fr] md:grid-cols-[auto_1fr]">
	<div class="hidden md:contents">
		<Sidebar {collapsed} onToggle={toggleCollapsed} />
	</div>

	<Topbar onMenu={() => (drawerOpen = true)} />

	<main id="content" class="min-w-0" tabindex="-1">
		<ApiDownBanner />
		<div class="mx-auto w-full max-w-shell px-4 py-6 md:px-6">
			{@render children()}
		</div>
	</main>

	<Drawer open={drawerOpen} onOpenChange={(open) => (drawerOpen = open)} label="Меню разделов">
		<div class="h-full">
			<Sidebar collapsed={false} onToggle={toggleCollapsed} />
		</div>
	</Drawer>
</div>

<ShortcutsModal open={shortcutsOpen} onOpenChange={(open) => (shortcutsOpen = open)} />
