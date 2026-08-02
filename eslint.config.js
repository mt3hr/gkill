import pluginVue from 'eslint-plugin-vue'
import vueTsEslintConfig from '@vue/eslint-config-typescript'
import pluginPlaywright from 'eslint-plugin-playwright'

export default [
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue,js,mjs}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**', 'src/server/**'],
  },
  ...pluginVue.configs['flat/essential'],
  ...vueTsEslintConfig(),
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'warn',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
          destructuredArrayIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-empty-object-type': 'error',
    },
  },
  {
    // E2Eテストで「静かに成功するテスト」が再発しないようにする。
    //
    // no-conditional-in-test:
    //   `if (await x.count() > 0) { ...本体... }` のように条件で包むと、
    //   要素が見つからないときに何も検証せずパスしてしまう。
    //   対象が見つかることを前提にする（見つからなければ失敗する）書き方に寄せる。
    //   待ってから掴みたい場合は crud-helpers の waitForKyouByText 等を使う。
    //
    // no-wait-for-timeout:
    //   固定時間のsleepは遅いマシンで足りずにフレークし、
    //   速いマシンでは無駄に待つ。expect(...).toBeVisible() 等の
    //   自動リトライする待機に置き換える。
    //
    // no-skipped-test は無効のまま。
    //   `test.skip(!alive, 'gkill server is not running')` は
    //   サーバ未起動時に飛ばすための正当な使い方。
    name: 'e2e/playwright',
    files: ['src/client/__tests__/e2e/**/*.ts'],
    plugins: { playwright: pluginPlaywright },
    rules: {
      'playwright/no-conditional-in-test': 'error',
      'playwright/no-wait-for-timeout': 'error',
      'playwright/no-skipped-test': 'off',
    },
  },
  {
    // 未移行のE2Eファイル。上のルールを warn に落としてある。
    //
    // 中核のCRUDフロー（add / edit / delete / notification / mi-operations）と
    // 共通ヘルパ（crud-helpers.ts）は hard assertion と web-first 待機に移行済みで、
    // 新しく追加するファイルには最初から error が効く。
    // 残りはここに列挙してあるぶんだけで、直したらこのリストから消すこと。
    // リストが空になったらこのブロックごと消せる。
    name: 'e2e/playwright-not-migrated',
    files: [
      'src/client/__tests__/e2e/auth.setup.ts',
      'src/client/__tests__/e2e/auth-flow.spec.ts',
      'src/client/__tests__/e2e/clipboard-save.spec.ts',
      'src/client/__tests__/e2e/dashboard.spec.ts',
      'src/client/__tests__/e2e/dialog-history.spec.ts',
      'src/client/__tests__/e2e/kftl-dialog.spec.ts',
      'src/client/__tests__/e2e/kyou-list.spec.ts',
      'src/client/__tests__/e2e/login.spec.ts',
      'src/client/__tests__/e2e/mi-board.spec.ts',
      'src/client/__tests__/e2e/misc-operations.spec.ts',
      'src/client/__tests__/e2e/mkfl.spec.ts',
      'src/client/__tests__/e2e/plaing.spec.ts',
      'src/client/__tests__/e2e/regist-first-account.spec.ts',
      'src/client/__tests__/e2e/regression-fixes.spec.ts',
      'src/client/__tests__/e2e/rykv.spec.ts',
      'src/client/__tests__/e2e/search-and-summary.spec.ts',
      'src/client/__tests__/e2e/server-config-crud.spec.ts',
      'src/client/__tests__/e2e/set-new-password.spec.ts',
      'src/client/__tests__/e2e/settings.spec.ts',
      'src/client/__tests__/e2e/shared-mi.spec.ts',
      'src/client/__tests__/e2e/user-config-crud.spec.ts',
      'src/client/__tests__/e2e/view-browse.spec.ts',
      'src/client/__tests__/e2e/view-history.spec.ts',
    ],
    rules: {
      'playwright/no-conditional-in-test': 'warn',
      'playwright/no-wait-for-timeout': 'warn',
    },
  },
]
