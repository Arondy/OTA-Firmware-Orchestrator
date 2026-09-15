<script lang="ts">
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon";
	import Icon from "./Icon.svelte";

	interface Token {
		text: string;
		cls: string;
	}

	/**
	 * Читалка JSON без подсветки-зависимости: ключи, строки и числа разбираются
	 * одним проходом и раскрашиваются токенами приложения. Внешнему раскрытию
	 * не мешает собственное сворачивание: содержимое открыто по умолчанию.
	 */
	let {
		value,
		label = "JSON",
		open = true,
	}: {
		value: unknown;
		label?: string;
		open?: boolean;
	} = $props();

	const TOKEN_PATTERN =
		/("(?:\\.|[^"\\])*")(\s*:)?|\b(?:true|false|null)\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/g;

	function tokenize(json: string): Token[] {
		const tokens: Token[] = [];
		let last = 0;

		for (const match of json.matchAll(TOKEN_PATTERN)) {
			const index = match.index ?? 0;
			if (index > last) tokens.push({ text: json.slice(last, index), cls: "text-fg-muted" });

			const [, quoted, colon, keyword] = match;
			if (quoted !== undefined) {
				tokens.push({
					text: quoted,
					cls: colon ? "text-fg-secondary" : "text-state-success",
				});
				if (colon) tokens.push({ text: colon, cls: "text-fg-muted" });
			} else if (keyword !== undefined) {
				tokens.push({ text: keyword, cls: "text-accent-text" });
			} else {
				tokens.push({ text: match[0], cls: "text-accent-text" });
			}
			last = index + match[0].length;
		}

		if (last < json.length) tokens.push({ text: json.slice(last), cls: "text-fg-muted" });
		return tokens;
	}

	const tokens = $derived.by(() => {
		try {
			return tokenize(JSON.stringify(value, null, 2) ?? "");
		} catch {
			return [{ text: "Значение не сериализуется в JSON", cls: "text-state-danger" }];
		}
	});
</script>

<details class="min-w-0" {open}>
	<summary
		class="flex cursor-pointer items-center gap-1 text-dense text-fg-muted hover:text-fg-secondary"
	>
		<Icon glyph={CaretRightIcon} size={16} />
		{label}
	</summary>
	<pre
		class="mt-1 max-h-72 overflow-auto rounded-chip bg-bg-inset p-2 font-mono text-dense leading-relaxed">{#each tokens as token, index (index)}<span
				class={token.cls}>{token.text}</span
			>{/each}</pre>
</details>
