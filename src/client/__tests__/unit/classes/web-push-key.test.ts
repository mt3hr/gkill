/**
 * Web Push の VAPID 公開鍵（URL-safe base64）を `pushManager.subscribe` に渡せる形へ変換する。
 *
 * 6つのページコンポーザブル（kftl / kyou / mi / mkfl / plaing / rykv）が
 * 同じ処理をそれぞれ持っていたのを1箇所へ寄せた関数なので、
 * ここが壊れると6画面すべてでプッシュ購読が同時に失敗する。
 * しかも失敗は `subscribe()` の例外としてしか出ないため、
 * 「なぜか通知だけ来ない」という形で表に出る。
 *
 * 変換の中身は3つで、どれも取り違えやすい:
 *   1. URL-safe → 標準 base64（`-`→`+`, `_`→`/`）
 *   2. `=` パディングの補完（長さ % 4 が 2 / 3 のときだけ足りない）
 *   3. バイト列化
 *
 * **落ちるのは 1 と 3 だけ。** `atob` は未パディングの入力を受け付ける
 * （WHATWG の仕様上、失敗するのは長さ % 4 == 1 のときだけ）ので、
 * 2 を消しても出力は変わらず、どのテストも落ちない ―― 実測で確認済み。
 * ここでのパディングは「入力が既に `=` 付きでも壊さない」ことの保証と、
 * 仕様変更への保険であって、テストで固定できる性質のものではない。
 * 下のパディング関連のケースは**出力が正しいこと**を見ているのであって、
 * パディング処理そのものを固定しているわけではない。
 */
import { describe, expect, test } from 'vitest'
import { url_base64_to_uint8_array } from '@/classes/web-push-key'

/** 期待値を作るための素直な実装（標準 base64 → バイト列） */
function bytes_of_standard_base64(standard: string): Array<number> {
    const raw = atob(standard)
    return Array.from(raw, c => c.charCodeAt(0))
}

describe('url_base64_to_uint8_array', () => {
    test('URL-safe の記号を標準 base64 に戻す', () => {
        // 0xFB 0xFF 0xBE は標準 base64 で "+/++"、URL-safe では "-_--"
        const url_safe = '-_--'
        expect(Array.from(url_base64_to_uint8_array(url_safe)))
            .toEqual(bytes_of_standard_base64('+/++'))
    })

    test.each([
        // [ラベル, パディング無しのURL-safe文字列, 長さ%4]
        ['パディング不要（%4 == 0）', 'AAAA', 0],
        ['= を2つ補う（%4 == 2）', 'AA', 2],
        ['= を1つ補う（%4 == 3）', 'AAA', 3],
    ])('%s', (_label, without_padding, mod) => {
        expect(without_padding.length % 4).toBe(mod)
        const padded = without_padding + '='.repeat((4 - mod) % 4)
        expect(Array.from(url_base64_to_uint8_array(without_padding)))
            .toEqual(bytes_of_standard_base64(padded))
    })

    test('すでに = が付いていても壊さない', () => {
        expect(Array.from(url_base64_to_uint8_array('AA=='))).toEqual([0])
    })

    test('実際の長さのVAPID公開鍵（65バイト）を変換できる', () => {
        // VAPID の公開鍵は非圧縮のP-256点で 65バイト。base64 では 87文字 + パディング1
        const bytes = new Uint8Array(65)
        for (let i = 0; i < bytes.length; i++) {
            bytes[i] = (i * 7 + 3) % 256
        }
        let binary = ''
        for (const b of bytes) {
            binary += String.fromCharCode(b)
        }
        const url_safe = btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
        expect(url_safe.length % 4).toBe(3) // パディング補完が要る長さであること

        const got = url_base64_to_uint8_array(url_safe)
        expect(got.length).toBe(65)
        expect(Array.from(got)).toEqual(Array.from(bytes))
    })

    test('戻り値は BufferSource として使える Uint8Array', () => {
        // 型注釈を Uint8Array<ArrayBuffer> にしてあるのは
        // applicationServerKey へ渡すところでキャストを不要にするため。
        // 実行時にも ArrayBuffer 実体であることを確かめておく
        const got = url_base64_to_uint8_array('AAAA')
        expect(got).toBeInstanceOf(Uint8Array)
        expect(got.buffer).toBeInstanceOf(ArrayBuffer)
    })

    test('空文字は空のバイト列', () => {
        expect(url_base64_to_uint8_array('').length).toBe(0)
    })
})
