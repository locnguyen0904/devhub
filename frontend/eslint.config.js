import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "src/shared/types/api.ts"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      // strictTypeChecked is what enforces the TypeScript rules in CLAUDE.md §4
      // (no any, no unnecessary assertions) — the untyped preset cannot see them.
      ...tseslint.configs.strictTypeChecked,
      // configs.flat.* is the flat-config shape; configs["recommended-latest"]
      // is still the legacy eslintrc one and ESLint 10 rejects it.
      reactHooks.configs.flat.recommended,
    ],
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: { "react-refresh": reactRefresh },
    rules: {
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      // Interfaces for object shapes, type aliases for unions — Google TS Style
      // Guide. The choice matters less than being consistent, so let the linter
      // hold the line instead of code review.
      "@typescript-eslint/consistent-type-definitions": ["error", "interface"],
      "@typescript-eslint/no-explicit-any": "error",
    },
  },
  {
    files: ["scripts/**/*.mjs", "eslint.config.js"],
    languageOptions: { globals: globals.node },
    extends: [tseslint.configs.disableTypeChecked],
  },
);
