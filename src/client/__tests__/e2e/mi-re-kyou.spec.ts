import { test, expect } from '@playwright/test'
import type { APIRequestContext, Page } from '@playwright/test'
import { checkGkillServer } from './check-server'
import { makeUniqueLabel } from './crud-helpers'

// MiReKyou（既存Kyouのタスク化）をAPI経由で通しで確認する。
// UIクリックだと板のドラッグやダイアログ待ちでフレークしやすいので、
// 実サーバのAPIを直接叩いて「作る→Mi画面の検索に出る→更新できる→
// 対象が消えたら出なくなる」まで検証する。

test.beforeAll(async () => {
  const alive = await checkGkillServer()
  test.skip(!alive, 'gkill server is not running')
})

function nowIso(): string {
  return new Date().toISOString()
}

function makeMeta(id: string) {
  return {
    is_deleted: false,
    id,
    related_time: nowIso(),
    create_app: 'e2e',
    create_device: 'e2e-device',
    create_time: nowIso(),
    create_user: 'admin',
    update_app: 'e2e',
    update_device: 'e2e-device',
    update_time: nowIso(),
    update_user: 'admin',
  }
}

/**
 * セッションIDを取得する。
 * gkillのAPIはCookieではなくリクエストボディの session_id で認証するので
 * (auth_middleware.go の sessionPeek)、Cookieから読み出して各リクエストへ載せる。
 */
async function getSessionID(page: Page): Promise<string> {
  const cookies = await page.context().cookies()
  const cookie = cookies.find((c) => c.name === 'gkill_session_id')
  expect(cookie, 'gkill_session_id cookie が見つからない').toBeTruthy()
  return cookie!.value
}

async function postJson(request: APIRequestContext, sessionID: string, path: string, body: Record<string, unknown>) {
  const res = await request.post(path, { data: { ...body, session_id: sessionID } })
  expect(res.ok(), `${path} should return 2xx`).toBeTruthy()
  return await res.json()
}

/** 対象のKmemoを作る。生成したIDを返す */
async function addKmemo(request: APIRequestContext, sessionID: string, content: string): Promise<string> {
  const id = crypto.randomUUID()
  const json = await postJson(request, sessionID, '/api/add_kmemo', {
    kmemo: { ...makeMeta(id), content },
    want_response_kyou: true,
  })
  expect(json.errors ?? [], `add_kmemo errors: ${JSON.stringify(json.errors)}`).toHaveLength(0)
  return id
}

/** MiReKyouを作る。生成したIDを返す */
async function addMiReKyou(request: APIRequestContext, sessionID: string, targetID: string, boardName: string): Promise<string> {
  const id = crypto.randomUUID()
  const json = await postJson(request, sessionID, '/api/add_mirekyou', {
    mirekyou: {
      ...makeMeta(id),
      target_id: targetID,
      is_checked: false,
      board_name: boardName,
      limit_time: null,
      estimate_start_time: null,
      estimate_end_time: null,
    },
    want_response_kyou: true,
  })
  expect(json.errors ?? [], `add_mirekyou errors: ${JSON.stringify(json.errors)}`).toHaveLength(0)
  return id
}

/** Mi画面と同じ条件(for_mi)で検索する */
async function findForMi(request: APIRequestContext, sessionID: string, boardName: string) {
  const json = await postJson(request, sessionID, '/api/get_kyous', {
    query: {
      for_mi: true,
      use_mi_board_name: true,
      mi_board_name: boardName,
      mi_check_state: 'all',
      mi_sort_type: 'create_time',
      include_create_mi: true,
      include_check_mi: true,
      include_limit_mi: true,
      include_start_mi: true,
      include_end_mi: true,
      only_latest_data: true,
    },
  })
  expect(json.errors ?? [], `get_kyous errors: ${JSON.stringify(json.errors)}`).toHaveLength(0)
  return (json.kyous ?? []) as Array<{ id: string, data_type: string }>
}

test.describe('MiReKyou (既存Kyouのタスク化)', () => {
  test('Kmemoをタスク化するとMi画面の検索に出る', async ({ page }) => {
    const request = page.request
    const sessionID = await getSessionID(page)
    const board = makeUniqueLabel('mirekyou_board')
    const content = makeUniqueLabel('mirekyou_target')

    const targetID = await addKmemo(request, sessionID, content)
    const mirekyouID = await addMiReKyou(request, sessionID, targetID, board)

    const kyous = await findForMi(request, sessionID, board)
    const found = kyous.find((k) => k.id === mirekyouID)
    expect(found, `MiReKyou ${mirekyouID} がMi画面の検索結果にない`).toBeTruthy()
    // クライアントが typed_mirekyou を引けるよう mirekyou_ 接頭辞になっていること
    expect(found!.data_type.startsWith('mirekyou_')).toBeTruthy()
  })

  test('get_mirekyouでtarget_idとボード名が引ける', async ({ page }) => {
    const request = page.request
    const sessionID = await getSessionID(page)
    const board = makeUniqueLabel('mirekyou_board')
    const targetID = await addKmemo(request, sessionID, makeUniqueLabel('mirekyou_target'))
    const mirekyouID = await addMiReKyou(request, sessionID, targetID, board)

    const json = await postJson(request, sessionID, '/api/get_mirekyou', { id: mirekyouID })
    expect(json.errors ?? []).toHaveLength(0)
    const histories = json.mirekyou_histories as Array<Record<string, unknown>>
    expect(histories.length).toBeGreaterThan(0)
    expect(histories[0].target_id).toBe(targetID)
    expect(histories[0].board_name).toBe(board)
    // MiReKyouはタイトルを持たない
    expect(histories[0].title).toBeUndefined()
  })

  test('チェック状態を更新できる', async ({ page }) => {
    const request = page.request
    const sessionID = await getSessionID(page)
    const board = makeUniqueLabel('mirekyou_board')
    const targetID = await addKmemo(request, sessionID, makeUniqueLabel('mirekyou_target'))
    const mirekyouID = await addMiReKyou(request, sessionID, targetID, board)

    // 追加と同じ秒だと UPDATE_TIME が並んで最新版が一意に定まらないので、
    // 明示的に後の時刻を入れる（時刻は秒精度で保存される）
    const laterUpdateTime = new Date(Date.now() + 5000).toISOString()
    const updateJson = await postJson(request, sessionID, '/api/update_mirekyou', {
      mirekyou: {
        ...makeMeta(mirekyouID),
        update_time: laterUpdateTime,
        target_id: targetID,
        is_checked: true,
        board_name: board,
        limit_time: null,
        estimate_start_time: null,
        estimate_end_time: null,
      },
      want_response_kyou: true,
    })
    expect(updateJson.errors ?? [], `update_mirekyou errors: ${JSON.stringify(updateJson.errors)}`).toHaveLength(0)

    const json = await postJson(request, sessionID, '/api/get_mirekyou', { id: mirekyouID })
    const histories = json.mirekyou_histories as Array<Record<string, unknown>>
    // 最新版がチェック済みになっていること
    const latest = histories.reduce((a, b) =>
      new Date(a.update_time as string) > new Date(b.update_time as string) ? a : b)
    expect(latest.is_checked).toBe(true)
  })

  test('MiReKyouだけのボードもボード一覧に出る', async ({ page }) => {
    const request = page.request
    const sessionID = await getSessionID(page)
    const board = makeUniqueLabel('mirekyou_onlyboard')
    const targetID = await addKmemo(request, sessionID, makeUniqueLabel('mirekyou_target'))
    await addMiReKyou(request, sessionID, targetID, board)

    const json = await postJson(request, sessionID, '/api/get_mi_board_list', {})
    expect(json.errors ?? []).toHaveLength(0)
    expect(json.boards as string[]).toContain(board)
  })

  test('対象Kyouを削除するとMi画面から消える', async ({ page }) => {
    const request = page.request
    const sessionID = await getSessionID(page)
    const board = makeUniqueLabel('mirekyou_board')
    const content = makeUniqueLabel('mirekyou_target')

    const targetID = await addKmemo(request, sessionID, content)
    const mirekyouID = await addMiReKyou(request, sessionID, targetID, board)

    // 作成直後は見えている
    const before = await findForMi(request, sessionID, board)
    expect(before.some((k) => k.id === mirekyouID)).toBeTruthy()

    // 対象のKmemoを論理削除する
    const deleteJson = await postJson(request, sessionID, '/api/update_kmemo', {
      kmemo: { ...makeMeta(targetID), content, is_deleted: true },
      want_response_kyou: true,
    })
    expect(deleteJson.errors ?? [], `update_kmemo errors: ${JSON.stringify(deleteJson.errors)}`).toHaveLength(0)

    // ターゲット解決フィルタで検索から落ちる
    const after = await findForMi(request, sessionID, board)
    expect(after.some((k) => k.id === mirekyouID),
      'ターゲットが削除されたMiReKyouがMi画面に残っている').toBeFalsy()
  })
})
