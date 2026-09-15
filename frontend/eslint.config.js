import js from "@eslint/js";
import tseslint from "typescript-eslint";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import prettier from "eslint-config-prettier";

export default tseslint.config(
	{
		ignores: [
			"build/**",
			".svelte-kit/**",
			"node_modules/**",
			// Генерируется из ota-orchestrator/api/openapi.yaml
			"src/lib/api/schema.d.ts",
		],
	},
	js.configs.recommended,
	...tseslint.configs.recommended,
	...svelte.configs["flat/prettier"],
	prettier,
	{
		languageOptions: {
			globals: { ...globals.browser, __APP_VERSION__: "readonly", __BUILD_TIME__: "readonly" },
			ecmaVersion: "latest",
			sourceType: "module",
		},
		rules: {
			eqeqeq: ["error", "always"],
			"no-alert": "error",
			"no-console": "error",
			"no-var": "error",
			"@typescript-eslint/no-explicit-any": "error",
			"@typescript-eslint/no-non-null-assertion": "error",
			"@typescript-eslint/ban-ts-comment": "error",
			"@typescript-eslint/consistent-type-imports": [
				"error",
				{ prefer: "type-imports", fixStyle: "separate-type-imports" },
			],
			"@typescript-eslint/no-unused-vars": [
				"error",
				{ argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
			],
			"svelte/no-at-html-tags": "error",
			"svelte/require-each-key": "error",
		},
	},
	{
		files: ["**/*.svelte", "**/*.svelte.ts", "**/*.svelte.js"],
		languageOptions: {
			globals: { ...globals.browser, __APP_VERSION__: "readonly", __BUILD_TIME__: "readonly" },
			parserOptions: {
				parser: tseslint.parser,
				extraFileExtensions: [".svelte"],
			},
		},
	},
	{
		// Сервисные скрипты и конфиги выполняются в Node, `console` там уместен.
		files: ["scripts/**/*.mjs", "*.config.ts", "*.config.js"],
		languageOptions: {
			globals: { ...globals.node },
		},
		rules: {
			"no-console": "off",
		},
	},
);
