import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

// CI only ever lints the files changed in a diff (see .github/workflows/ci.yml),
// but ESLint still parses/checks the whole file it's given, not just the
// changed lines — so the rule set itself has to stay lenient too, or a
// one-line edit to an existing file surfaces hundreds of pre-existing
// warnings unrelated to the change. This codebase's templates are also
// deliberately written dense/single-line (see AGENTS.md conventions), which
// vue's "recommended" preset's formatting rules (interpolation spacing,
// self-closing-tag spacing, etc.) actively fight — so this uses vue's
// "essential" preset (bugs only, no formatting/style rules) rather than
// "recommended" or "strict".
export default tseslint.config(
  { ignores: ['dist/**', 'dev-dist/**', 'node_modules/**'] },
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: { parser: tseslint.parser },
    },
  },
  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'vue/multi-word-component-names': 'off',
      'vue/require-default-prop': 'off',
      'vue/html-self-closing': 'off',
      'vue/max-attributes-per-line': 'off',
      'vue/singleline-html-element-content-newline': 'off',
    },
  },
)
