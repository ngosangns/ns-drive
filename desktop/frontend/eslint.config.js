// @ts-check
const eslint = require("@eslint/js");
const tseslint = require("typescript-eslint");
const angular = require("angular-eslint");

module.exports = tseslint.config(
  {
    files: ["**/*.ts"],
    extends: [
      eslint.configs.recommended,
      ...tseslint.configs.recommended,
      ...tseslint.configs.stylistic,
      ...angular.configs.tsRecommended,
    ],
    processor: angular.processInlineTemplates,
    rules: {
      "@angular-eslint/directive-selector": [
        "error",
        {
          type: "attribute",
          prefix: "app",
          style: "camelCase",
        },
      ],
      "@angular-eslint/component-selector": [
        "error",
        {
          type: "element",
          prefix: "app",
          style: "kebab-case",
        },
      ],
      "prettier/prettier": [
        "error",
        {
          endOfLine: "lf",
          singleQuote: false,
          tabWidth: 4,
          useTabs: false,
          trailingComma: "es5",
          semi: true,
          bracketSpacing: true,
          arrowParens: "avoid",
          bracketSameLine: true,
          printWidth: 120,
        },
      ],
    },
  },
  {
    files: ["**/*.html"],
    extends: [
      ...angular.configs.templateRecommended,
      ...angular.configs.templateAccessibility,
    ],
    rules: {
      "prettier/prettier": [
        "error",
        {
          endOfLine: "lf",
          singleQuote: false,
          tabWidth: 4,
          useTabs: false,
          trailingComma: "es5",
          semi: true,
          bracketSpacing: true,
          arrowParens: "avoid",
          bracketSameLine: true,
          printWidth: 120,
        },
      ],
    },
  }
);
