import { describe, test, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import ja from '../../../../locales/ja.json'
import en from '../../../../locales/en.json'
import { error_hint_message_id, reason_tokens, error_kind_tokens } from '@/classes/api/message/error-hints'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

// ヒント文（何をすれば直るか）は error_kind / reason の機械語トークンから引く。
// トークンの語彙は Go 側（error_kind.go / error_reason.go）が正本で、TS 側の表と食い違うと
// 「サーバは reason を返しているのにヒントが出ない」が静かに起きる。ソースを突き合わせて固定する。

const GO_REASON_FILE = 'src/server/gkill/api/message/error_reason.go'
const GO_KIND_FILE = 'src/server/gkill/api/message/error_kind.go'

function go_string_consts(path: string, prefix: string): Set<string> {
  const src = readFileSync(path, 'utf-8')
  const re = new RegExp(`^\\s*${prefix}\\w+\\s*=\\s*"([a-z_]+)"`, 'gm')
  const tokens = new Set<string>()
  for (const m of src.matchAll(re)) {
    tokens.add(m[1])
  }
  return tokens
}

describe('error-hints', () => {
  test('reason の語彙が Go 側（error_reason.go の Reason* 定数）と一致する', () => {
    const go_tokens = go_string_consts(GO_REASON_FILE, 'Reason')
    expect(go_tokens.size, 'Go 側の定数が読めていない').toBeGreaterThan(0)
    expect(new Set(reason_tokens)).toEqual(go_tokens)
  })

  test('error_kind の語彙が Go 側（error_kind.go の ErrorKind* 定数）と一致する', () => {
    const go_tokens = go_string_consts(GO_KIND_FILE, 'ErrorKind')
    expect(go_tokens.size).toBeGreaterThan(0)
    expect(new Set(error_kind_tokens)).toEqual(go_tokens)
  })

  test('全トークンのヒントの i18n キーが ja / en の両方に存在する', () => {
    const ids = new Set<string>()
    for (const reason of reason_tokens) {
      ids.add(error_hint_message_id({ reason }))
    }
    for (const kind of error_kind_tokens) {
      const id = error_hint_message_id({ error_kind: kind })
      if (id !== '') {
        ids.add(id)
      }
    }
    ids.add(error_hint_message_id({ error_code: GkillErrorCodes.bad_response }))
    ids.add(error_hint_message_id({ error_code: GkillErrorCodes.unexpected_client_error }))
    expect(ids.size).toBeGreaterThan(20)
    for (const id of ids) {
      expect(id, 'キーが空').not.toBe('')
      expect((ja as Record<string, string>)[id], `ja に ${id} が無い`).toBeTruthy()
      expect((en as Record<string, string>)[id], `en に ${id} が無い`).toBeTruthy()
    }
  })

  test('reason が kind より優先され、未知の reason は kind へ落ちる', () => {
    expect(error_hint_message_id({ error_kind: 'server', reason: 'db_busy' })).toBe('ERROR_HINT_REASON_DB_BUSY')
    expect(error_hint_message_id({ error_kind: 'server', reason: 'unknown_future_token' })).toBe('ERROR_HINT_KIND_SERVER')
    expect(error_hint_message_id({ error_kind: 'server' })).toBe('ERROR_HINT_KIND_SERVER')
  })

  test('auth と kind 無し（クライアントの入力検証）にはヒントが無い', () => {
    expect(error_hint_message_id({ error_kind: 'auth' })).toBe('')
    expect(error_hint_message_id({ error_code: 'ERR900013' })).toBe('')
    expect(error_hint_message_id({})).toBe('')
  })

  test('クライアント生成のコードは自分のヒントを持つ', () => {
    expect(error_hint_message_id({ error_code: GkillErrorCodes.bad_response, error_kind: 'server' })).toBe('ERROR_HINT_BAD_RESPONSE')
    expect(error_hint_message_id({ error_code: GkillErrorCodes.unexpected_client_error })).toBe('ERROR_HINT_UNEXPECTED_CLIENT_ERROR')
  })
})
