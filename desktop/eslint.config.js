import { fixupConfigRules, fixupPluginRules } from "@eslint/compat"
import react from "eslint-plugin-react"
import typescriptEslint from "@typescript-eslint/eslint-plugin"
import tanstackQuery from "@tanstack/eslint-plugin-query"
import globals from "globals"
import tsParser from "@typescript-eslint/parser"
import path from "node:path"
import { fileURLToPath } from "node:url"
import js from "@eslint/js"
import { FlatCompat } from "@eslint/eslintrc"
import reactRefresh from "eslint-plugin-react-refresh"

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all,
})

export default [
  {
    ignores: ["dist/**/*", "src-tauri/**/*", "public/**/*", "src/gen/**/*"],
  },
  ...fixupConfigRules(
    compat.extends(
      "eslint:recommended",
      "plugin:react/recommended",
      "plugin:@typescript-eslint/eslint-recommended",
      "plugin:react-hooks/recommended",
      "prettier",
      "plugin:@tanstack/eslint-plugin-query/recommended"
    )
  ),
  reactRefresh.configs.recommended,
  {
    plugins: {
      react: fixupPluginRules(react),
      "@typescript-eslint": fixupPluginRules(typescriptEslint),
      "@tanstack/query": fixupPluginRules(tanstackQuery),
    },

    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
        Atomics: "readonly",
        SharedArrayBuffer: "readonly",
      },

      parser: tsParser,
      ecmaVersion: 5,

      parserOptions: {
        project: true,
        tsConfigRootDir: __dirname,
      },
    },

    settings: {
      react: {
        version: "detect",
      },
    },
    rules: {
      "react/react-in-jsx-scope": "off",
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["error"],

      "padding-line-between-statements": [
        "warn",
        {
          blankLine: "always",
          prev: "*",
          next: "return",
        },
      ],

      "no-warning-comments": [
        "error",
        {
          terms: ["fixme"],
          location: "start",
        },
      ],

      "@typescript-eslint/no-unnecessary-condition": [
        "warn",
        {
          allowConstantLoopConditions: true,
        },
      ],

      "@typescript-eslint/naming-convention": [
        "error",
        {
          selector: ["typeParameter", "typeAlias"],
          format: ["PascalCase"],
          prefix: ["T"],
        },
        {
          selector: ["interface"],
          format: ["PascalCase"],
          prefix: ["I"],
        },
        {
          selector: ["enum"],
          format: ["PascalCase"],
          prefix: ["E"],
        },
      ],
    },
  },
]
