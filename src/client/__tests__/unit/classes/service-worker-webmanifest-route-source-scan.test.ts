/**
 * PWA の manifest 専用ルートが precache より先に登録されていることの固定。
 *
 * Chrome が manifest を取りに来る fetch も Service Worker を通る。manifest は
 * vite-plugin-pwa が additionalManifestEntries で precache へ入れており
 * （vite.config.ts の globIgnores では外せない）、専用ルートが無いか、あっても
 * `precacheAndRoute` より後ろにあると、Workbox のルータは登録順に最初に一致した
 * ものを使うので precache の古いコピーが返り続ける。
 *
 * そうなると theme_color / icons / name を変えてサーバへ配っても端末へ永久に届かず、
 * ホーム画面から消して入れ直しても直らない（再インストール時の取得も
 * Service Worker が答えるため）。Android エミュレータでの実測では、専用ルートが
 * 無い版は SW 有効化後の再読み込みで manifest のネットワーク取得が **0件**、
 * 入れた版は毎回ネットワークまで届いた。
 *
 * ビルドもlintも通り、画面上は何も起きないので、ここでしか気付けない。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'

// import.meta.url は vitest の変換後は file スキームにならないので使えない。
// package.json を目印に上へ辿ってリポジトリルートを決める
function find_repo_root(): string {
    let dir = process.cwd()
    for (let i = 0; i < 10; i++) {
        if (existsSync(join(dir, 'package.json')) && existsSync(join(dir, 'src', 'client'))) {
            return dir
        }
        const parent = dirname(dir)
        if (parent === dir) {
            break
        }
        dir = parent
    }
    throw new Error(`リポジトリルートが見つからない: cwd=${process.cwd()}`)
}

const source = readFileSync(join(find_repo_root(), 'src', 'client', 'serviceWorker.ts'), 'utf8')

describe('serviceWorker.ts の manifest ルート', () => {
    it('manifest 専用の registerRoute がある', () => {
        expect(
            source.includes("url.pathname === '/manifest.webmanifest'"),
            'manifest 専用ルートが無い。precache の古い manifest が返り続ける',
        ).toBe(true)
    })

    it('manifest ルートは precacheAndRoute より先に登録されている', () => {
        const manifest_route_at = source.indexOf("url.pathname === '/manifest.webmanifest'")
        const precache_at = source.indexOf('precacheAndRoute(')
        expect(manifest_route_at, 'manifest 専用ルートが見つからない').toBeGreaterThan(-1)
        expect(precache_at, 'precacheAndRoute が見つからない').toBeGreaterThan(-1)
        expect(
            manifest_route_at,
            'Workbox は登録順に最初に一致したルートを使う。precacheAndRoute より後ろだと precache が先に持っていく',
        ).toBeLessThan(precache_at)
    })

    it('manifest ルートはネットワークを先に見る戦略になっている', () => {
        // CacheFirst にすると precache に入れているのと変わらなくなる
        const around = source.slice(
            source.indexOf("url.pathname === '/manifest.webmanifest'"),
            source.indexOf('precacheAndRoute('),
        )
        expect(
            around.includes('NetworkFirst') || around.includes('NetworkOnly'),
            'manifest はネットワークを先に見ること（CacheFirst では古いままになる）',
        ).toBe(true)
    })
})
