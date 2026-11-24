/**
 * 地図を描いている .vue が、表示言語と地域をローダへ渡しているかを走査する。
 *
 * 渡さないと読み込み URL に `region=undefined&language=undefined` が乗り、
 * 言語が無効なので**地図のラベルがアプリの表示言語に従わなくなる**。
 * ローダは渡されたオブジェクトのキーを `for...in` で回して `URLSearchParams.set` するので、
 * 値が `undefined` のキーも文字列 `"undefined"` になる。`GoogleMap` の props に既定値は無い。
 *
 * **さらに、ローダは `setOptions` の最初の1回だけを採る**（2回目以降は警告して無視）。
 * つまり「最初にマウントされた地図の props がプロセス全体の URL を決める」ので、
 * 片方だけ直すと、どちらが先に描かれたかで結果が変わる。
 * 3枚目の地図が増えたときに黙って戻るのを止める網はここだけ。
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
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

const repo_root = find_repo_root()
const pages_root = join(repo_root, 'src', 'client', 'pages')

function list_vue_files(dir: string): Array<string> {
    const found = new Array<string>()
    for (const entry of readdirSync(dir)) {
        const path = join(dir, entry)
        if (statSync(path).isDirectory()) {
            found.push(...list_vue_files(path))
            continue
        }
        if (path.endsWith('.vue')) {
            found.push(path)
        }
    }
    return found
}

function to_repo_path(path: string): string {
    return relative(repo_root, path).split(sep).join('/')
}

/** `<GoogleMap` を書いている .vue を集める */
function collect_map_views(): Array<{ repo_path: string, source: string }> {
    const found = new Array<{ repo_path: string, source: string }>()
    for (const path of list_vue_files(pages_root)) {
        const source = readFileSync(path, 'utf8')
        if (source.includes('<GoogleMap')) {
            found.push({ repo_path: to_repo_path(path), source: source })
        }
    }
    return found
}

describe('地図のローダへ渡す表示言語と地域の走査', () => {
    const map_views = collect_map_views()

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('地図を描いているビューを見つけられている', () => {
        expect(map_views.map(view => view.repo_path).sort()).toEqual([
            'src/client/pages/views/gps-log-map.vue',
            'src/client/pages/views/map-query.vue',
        ])
    })

    it.each([':language="google_map_language"', ':region="google_map_region"'])(
        '%s を渡している',
        (attribute) => {
            const missing = map_views
                .filter(view => !view.source.includes(attribute))
                .map(view => view.repo_path)
            expect(
                missing,
                '渡さないと地図の URL に undefined が乗る（最初にマウントされた地図の値が全体を決める）',
            ).toEqual([])
        },
    )

    // 値はビューの composable（`.vue` の script には計算を置かない）が共有の1実装から取る
    it('値は共有の1実装（google-map-locale.ts）から、ビューの composable 経由で取っている', () => {
        const missing = map_views
            .filter(view => {
                const composable = /from '@\/classes\/(use-[a-z0-9-]+)'/.exec(view.source)
                if (!composable) {
                    return true
                }
                const composable_source = readFileSync(join(repo_root, 'src', 'client', 'classes', `${composable[1]}.ts`), 'utf8')
                return !composable_source.includes("from '@/classes/google-map-locale'")
                    || !composable_source.includes('google_map_language,')
                    || !composable_source.includes('google_map_region,')
            })
            .map(view => view.repo_path)
        expect(
            missing,
            '地図ごとに別の値を書くと、どちらが先に描かれたかで URL が変わる',
        ).toEqual([])
    })
})
