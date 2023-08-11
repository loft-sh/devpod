module.exports = {
  root: true,
  env: {
    browser: true,
    es6: true,
    node: true,
  },
  extends: [
    "eslint:recommended",
    "plugin:react/recommended",
    "plugin:@typescript-eslint/eslint-recommended",
    "plugin:react-hooks/recommended",
    "prettier",
    "plugin:@tanstack/eslint-plugin-query/recommended",
  ],
  globals: {
    Atomics: "readonly",
    SharedArrayBuffer: "readonly",
  },
  parser: "@typescript-eslint/parser",
  parserOptions: {
    project: ["./tsconfig.json"],
  },
  plugins: ["react", "@typescript-eslint", "@tanstack/query"],
  settings: {
    react: {
      version: "detect",
    },
  },
  rules: {
    "react/react-in-jsx-scope": "off",
    "no-unused-vars": "off", // overridden by `@typescript-eslint`
    "@typescript-eslint/no-unused-vars": ["error"],
    "padding-line-between-statements": ["warn", { blankLine: "always", prev: "*", next: "return" }],
    "no-warning-comments": ["error", { terms: ["fixme"], location: "start" }],
    "@typescript-eslint/no-unnecessary-condition": ["warn", { allowConstantLoopConditions: true }],
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
  ignorePatterns: ["dist/**/*", "src-tauri/**/*", "public/**/*", "src/gen/**/*"],
}
