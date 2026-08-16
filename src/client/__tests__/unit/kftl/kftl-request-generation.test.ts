import { describe, test, expect, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTLKmemoRequest } from '@/classes/kftl/kftl_kmemo/kftl-kmemo-request'
import { KFTLMiRequest } from '@/classes/kftl/kftl_mi/kftl-mi-request'
import { KFTLMiReKyouRequest } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-request'
import { KFTLNlogRequest } from '@/classes/kftl/kftl_nlog/kftl-nlog-request'
import type { KFTLRequest } from '@/classes/kftl/kftl-request'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

function pick_mi_re_kyou_request(requests: Array<KFTLRequest>): KFTLMiReKyouRequest {
  const found = requests.find((request) => request instanceof KFTLMiReKyouRequest)
  expect(found).toBeInstanceOf(KFTLMiReKyouRequest)
  return found as KFTLMiReKyouRequest
}

function pick_nlog_requests(requests: Array<KFTLRequest>): Array<KFTLNlogRequest> {
  return requests.filter((request) => request instanceof KFTLNlogRequest) as Array<KFTLNlogRequest>
}

function pick_kmemo_request(requests: Array<KFTLRequest>): KFTLKmemoRequest {
  const found = requests.find((request) => request instanceof KFTLKmemoRequest)
  expect(found).toBeInstanceOf(KFTLKmemoRequest)
  return found as KFTLKmemoRequest
}

describe('KFTL Request Generation', () => {
  // generate_requests() depends on GkillAPI.get_gkill_api().generate_uuid()
  // which uses crypto.getRandomValues — available in jsdom.

  describe('single line text (kmemo)', () => {
    test('single line of plain text generates one kmemo request', async () => {
      const stmt = new KFTLStatement('テストメモ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      expect(requests[0]).toBeInstanceOf(KFTLKmemoRequest)
    })

    test('kmemo request contains the text content', async () => {
      const stmt = new KFTLStatement('メモの内容')
      const requests = await stmt.generate_requests()
      const req = requests[0] as KFTLKmemoRequest
      // KFTLKmemoRequest stores content via add_kmemo_line; verify via request_id existing
      expect(req.get_request_id()).toBeTruthy()
    })
  })

  describe('multi-line kmemo', () => {
    test('multiple plain text lines generate one kmemo request', async () => {
      const stmt = new KFTLStatement('一行目\n二行目\n三行目')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      expect(requests[0]).toBeInstanceOf(KFTLKmemoRequest)
    })
  })

  describe('tags', () => {
    test('tag line adds a tag to the preceding request', async () => {
      const stmt = new KFTLStatement('テストメモ\n。タグ名')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      const tags = requests[0].get_tags()
      expect(tags).toContain('タグ名')
    })

    test('multiple tags accumulate on the same request', async () => {
      const stmt = new KFTLStatement('テストメモ\n。タグ1\n。タグ2')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      const tags = requests[0].get_tags()
      expect(tags).toContain('タグ1')
      expect(tags).toContain('タグ2')
    })
  })

  describe('split (、) generates multiple requests', () => {
    test('split separator creates two separate requests', async () => {
      const stmt = new KFTLStatement('最初のメモ\n、\n次のメモ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
    })

    test('each request after split is independent', async () => {
      const stmt = new KFTLStatement('メモA\n。タグA\n、\nメモB\n。タグB')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      expect(requests[0].get_tags()).toContain('タグA')
      expect(requests[0].get_tags()).not.toContain('タグB')
      expect(requests[1].get_tags()).toContain('タグB')
      expect(requests[1].get_tags()).not.toContain('タグA')
    })
  })

  describe('split and next second (、、)', () => {
    test('double-split separator creates two separate requests', async () => {
      const stmt = new KFTLStatement('最初のメモ\n、、\n次のメモ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
    })
  })

  describe('empty and minimal inputs', () => {
    test('empty text generates one request (empty kmemo)', async () => {
      const stmt = new KFTLStatement('')
      const requests = await stmt.generate_requests()
      // Even empty text produces a kmemo request (content will be empty)
      expect(requests.length).toBe(1)
    })

    test('whitespace-only text generates one request', async () => {
      const stmt = new KFTLStatement('  ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
    })
  })

  describe('request IDs', () => {
    test('each request has a non-empty request_id', async () => {
      const stmt = new KFTLStatement('メモA\n、\nメモB')
      const requests = await stmt.generate_requests()
      for (const req of requests) {
        expect(req.get_request_id()).toBeTruthy()
        expect(req.get_request_id().length).toBeGreaterThan(0)
      }
    })

    test('different requests have different IDs', async () => {
      const stmt = new KFTLStatement('メモA\n、\nメモB')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      expect(requests[0].get_request_id()).not.toBe(requests[1].get_request_id())
    })
  })

  describe('related time', () => {
    test('request without related time prefix uses current time', async () => {
      const stmt = new KFTLStatement('テストメモ')
      const requests = await stmt.generate_requests()
      const relTime = requests[0].get_related_time()
      expect(relTime).toBeInstanceOf(Date)
    })

    test('related time prefix sets a custom time on the request', async () => {
      const stmt = new KFTLStatement('？2025-01-15 10:00:00\nテストメモ')
      const requests = await stmt.generate_requests()
      const relTime = requests[0].get_related_time()
      expect(relTime).toBeInstanceOf(Date)
      // The related time should be set to 2025-01-15
      if (relTime) {
        expect(relTime.getFullYear()).toBe(2025)
        expect(relTime.getMonth()).toBe(0) // January
        expect(relTime.getDate()).toBe(15)
      }
    })
  })

  describe('get_invalid_line_indexs', () => {
    test('valid text returns empty invalid indexes', async () => {
      const stmt = new KFTLStatement('テストメモ')
      const invalids = await stmt.get_invalid_line_indexs()
      expect(invalids).toEqual([])
    })
  })

  // MCP(Go側パーサー)と同じASCIIプレフィックスでの入力
  describe('ASCII prefixes', () => {
    test('"#" tag line with "," separator adds both tags without prefix', async () => {
      const stmt = new KFTLStatement('テストメモ\n#tag1,tag2')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      const tags = requests[0].get_tags()
      expect(tags).toContain('tag1')
      expect(tags).toContain('tag2')
    })

    test('"#" tag line with "、" separator also splits', async () => {
      const stmt = new KFTLStatement('テストメモ\n#tag1、tag2')
      const requests = await stmt.generate_requests()
      const tags = requests[0].get_tags()
      expect(tags).toContain('tag1')
      expect(tags).toContain('tag2')
    })

    test('"?" related time line sets the parsed time', async () => {
      const stmt = new KFTLStatement('?2025-01-15 10:00:00\nテストメモ')
      const requests = await stmt.generate_requests()
      const relTime = requests[0].get_related_time()
      expect(relTime).toBeInstanceOf(Date)
      if (relTime) {
        expect(relTime.getFullYear()).toBe(2025)
        expect(relTime.getMonth()).toBe(0) // January
        expect(relTime.getDate()).toBe(15)
      }
    })

    test('"," split separator creates two separate requests', async () => {
      const stmt = new KFTLStatement('最初のメモ\n,\n次のメモ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
    })

    test('",," split separator creates two separate requests', async () => {
      const stmt = new KFTLStatement('最初のメモ\n,,\n次のメモ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
    })

    test('"--" text block fences generate one request', async () => {
      const stmt = new KFTLStatement('メモ\n--\nブロック内容\n--')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
    })

    test('"!" save character stops parsing of following lines', async () => {
      const stmt = new KFTLStatement('メモ\n!\n無視される行\n。無視されるタグ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      expect(requests[0].get_tags()).not.toContain('無視されるタグ')
    })

    test('"！" save character regression: stops parsing as before', async () => {
      const stmt = new KFTLStatement('メモ\n！\n無視される行\n。無視されるタグ')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      expect(requests[0].get_tags()).not.toContain('無視されるタグ')
    })
  })

  /**
   * リポストタスク(「～～」で開いて「～～」で閉じるブロック)。
   * 同じレコードで書いたKyouをタスク化するので、対象のidはバケツリレーされてきたもの、
   * MiReKyou自身のidは別に採番される。
   */
  describe('MiReKyou (～～)', () => {
    test('ーみ は今までどおり Mi のまま', async () => {
      const stmt = new KFTLStatement('ーみ\nタスク名')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(1)
      expect(requests[0]).toBeInstanceOf(KFTLMiRequest)
    })

    test('メモの後ろのブロックが MiReKyou のリクエストになる', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n仕事\n～～')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      expect(pick_mi_re_kyou_request(requests).get_mi_board_name()).toBe('仕事')
    })

    test('ASCII の ~~ でも同じリクエストになる', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n~~\n仕事\n~~')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      expect(pick_mi_re_kyou_request(requests).get_mi_board_name()).toBe('仕事')
    })

    // target_id を作り直してしまうと、対象を見失って保存されない
    test('target_id が直前のメモの request_id を指す', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n仕事\n～～')
      const requests = await stmt.generate_requests()
      const mi_re_kyou_request = pick_mi_re_kyou_request(requests)
      expect(mi_re_kyou_request.get_target_id()).toBe(pick_kmemo_request(requests).get_request_id())
    })

    test('MiReKyou 自身の request_id は対象の id とは別', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n仕事\n～～')
      const requests = await stmt.generate_requests()
      const mi_re_kyou_request = pick_mi_re_kyou_request(requests)
      expect(mi_re_kyou_request.get_request_id()).not.toBe(mi_re_kyou_request.get_target_id())
    })

    // ブロックを先に書いてあとからメモを書く並び。ブロックの各行がプロトタイプかどうかを
    // 次の行へ伝えていないと、メモが別のidを引き当てて対象が消える
    test('ブロックを先に書いてもメモと同じ target_id になる', async () => {
      const stmt = new KFTLStatement('～～\n仕事\n～～\n牛乳を買う')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      const mi_re_kyou_request = pick_mi_re_kyou_request(requests)
      expect(mi_re_kyou_request.get_target_id()).toBe(pick_kmemo_request(requests).get_request_id())
    })

    test('板名を書かなければ空のまま(既定の板へのフォールバックは do_request 側)', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n～～')
      const requests = await stmt.generate_requests()
      expect(pick_mi_re_kyou_request(requests).get_mi_board_name()).toBe('')
    })

    test('ブロックの中のタグは板名の前でも後でも MiReKyou に付く', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n。今日中\n仕事\n\n\n\n。重要\n～～')
      const requests = await stmt.generate_requests()
      const tags = pick_mi_re_kyou_request(requests).get_tags()
      expect(tags).toContain('今日中')
      expect(tags).toContain('重要')
    })

    test('閉じたあとのタグは対象のメモに付く', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n。今日中\n仕事\n～～\n。買い物')
      const requests = await stmt.generate_requests()
      expect(pick_kmemo_request(requests).get_tags()).toEqual(['買い物'])
      expect(pick_mi_re_kyou_request(requests).get_tags()).toEqual(['今日中'])
    })

    // 対象が無いMiReKyouは検索でターゲット解決に失敗して結果から落ちる。
    // 画面に出ないのに消せない行が残るので、書く前にエラーにする
    test('レコードに対象のKyouが無ければ do_request がエラーを返す', async () => {
      const stmt = new KFTLStatement('～～\n仕事\n～～')
      const requests = await stmt.generate_requests()
      const errors = await requests[0].do_request({} as unknown as GkillAPI, {} as unknown as ApplicationConfig)
      expect(errors.length).toBe(1)
      expect(errors[0].error_code).toBe('ERR900090')
    })

    test('タグだけのレコード(プロトタイプのみ)でも do_request がエラーを返す', async () => {
      const stmt = new KFTLStatement('。買い物\n～～\n仕事\n～～')
      const requests = await stmt.generate_requests()
      const mi_re_kyou_request = pick_mi_re_kyou_request(requests)
      const errors = await mi_re_kyou_request.do_request({} as unknown as GkillAPI, {} as unknown as ApplicationConfig)
      expect(errors.length).toBe(1)
      expect(errors[0].error_code).toBe('ERR900090')
    })

    // 飲み込むと、閉じ忘れたときにメモの本文が丸ごとタグになってしまう
    test('項目行を書き終えたあとの非タグ行はおかしな行になる', async () => {
      const stmt = new KFTLStatement('メモ\n～～\n仕事\n\n\n\nただの文\n～～')
      const invalids = await stmt.get_invalid_line_indexs()
      expect(invalids).toContain(6)
    })

    test('閉じ忘れると以降の行がおかしな行になる', async () => {
      const stmt = new KFTLStatement('メモ\n～～\n仕事\n\n\n\n。今日中\nつづきのメモ')
      const invalids = await stmt.get_invalid_line_indexs()
      expect(invalids).toContain(7)
    })

    test('きちんと閉じてあればおかしな行は無い', async () => {
      const stmt = new KFTLStatement('メモ\n～～\n。今日中\n仕事\n2025-03-20\n\n2025-03-22\n。重要\n～～\n。買い物')
      const invalids = await stmt.get_invalid_line_indexs()
      expect(invalids).toEqual([])
    })

    // MiReKyou を独立したリクエストにしている理由。基底の KFTLRequest に相乗りさせると
    // KFTLMiRequest の get_mi_board_name() の override に食われて、
    // リポストタスクの板名が未知板名の確認から漏れる
    test('同じレコードに ーみ と ～～ を書いても板名がそれぞれ返る', async () => {
      const stmt = new KFTLStatement('ーみ\nタスク名\nMiの板\n\n\n\n～～\nリポストの板\n～～')
      const requests = await stmt.generate_requests()
      const board_names = requests.map((request) => request.get_mi_board_name()).filter((name) => name !== '')
      expect(board_names.sort()).toEqual(['Miの板', 'リポストの板'])
    })

    test('閉じたあとは「、」で次のレコードに進める', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n～～\n仕事\n～～\n、\n次のメモ')
      const requests = await stmt.generate_requests()
      // メモ + リポストタスク + 次のメモ
      expect(requests.length).toBe(3)
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
      const mi_re_kyou_request = pick_mi_re_kyou_request(requests)
      const kmemo_requests = requests.filter((request) => request instanceof KFTLKmemoRequest)
      expect(kmemo_requests.length).toBe(2)
      // 対象は同じレコードのメモで、区切りの後のメモではない
      expect(mi_re_kyou_request.get_target_id()).toBe(kmemo_requests[0].get_request_id())
      expect(mi_re_kyou_request.get_target_id()).not.toBe(kmemo_requests[1].get_request_id())
    })

    test('波ダッシュ(U+301C)で書いても同じリクエストになる', async () => {
      const stmt = new KFTLStatement('牛乳を買う\n〜〜\n仕事\n〜〜')
      const requests = await stmt.generate_requests()
      expect(requests.length).toBe(2)
      expect(pick_mi_re_kyou_request(requests).get_mi_board_name()).toBe('仕事')
    })
  })

  /**
   * 支出(「ーん」のブロック)。支払い(品名と金額のペア)1組ごとに1つのリクエストになるので、
   * 金額の行のあとに書いたタグ・テキストは「直前の支払い」だけに付く。
   * 店名と関連時刻だけがブロック全体で共有される。
   */
  describe('Nlog (ーん)', () => {
    test('支払いの数だけリクエストができ、店名は共有される', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\nお茶\n120')
      const requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(requests)
      expect(nlogs.length).toBe(2)
      expect(nlogs.map((nlog) => nlog.title)).toEqual(['おにぎり', 'お茶'])
      expect(nlogs.map((nlog) => nlog.amount)).toEqual([150, 120])
      expect(nlogs.map((nlog) => nlog.shop_name)).toEqual(['コンビニ', 'コンビニ'])
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('支払いごとに別の request_id を持つ', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\nお茶\n120')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      expect(nlogs[0].get_request_id()).not.toBe(nlogs[1].get_request_id())
    })

    test('金額の行のあとのタグは直前の支払いだけに付く', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n。食費\nお茶\n120\n。飲み物')
      const stmt_requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(stmt_requests)
      expect(nlogs.length).toBe(2)
      expect(nlogs[0].get_tags()).toEqual(['食費'])
      expect(nlogs[1].get_tags()).toEqual(['飲み物'])
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('1行に複数タグを書いても直前の支払いだけに付く', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n。食費、朝食\nお茶\n120')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      expect(nlogs[0].get_tags()).toEqual(['食費', '朝食'])
      expect(nlogs[1].get_tags()).toEqual([])
    })

    // タグ行でブロックが切れると、以降の品名行が拾われずおかしな行になっていた
    test('タグ行のあとも品名行を続けられる', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n。食費\n。朝食\nお茶\n120')
      const stmt_requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(stmt_requests)
      expect(nlogs.length).toBe(2)
      expect(nlogs[0].get_tags()).toEqual(['食費', '朝食'])
      expect(nlogs[1].title).toBe('お茶')
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('金額の行のあとのテキストブロックは直前の支払いに付き、閉じたあとも続けられる', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\nーー\n朝ごはん用\nーー\nお茶\n120')
      const stmt_requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(stmt_requests)
      expect(nlogs.length).toBe(2)
      expect(nlogs[0].get_texts()).toEqual(['朝ごはん用'])
      expect(nlogs[1].get_texts()).toEqual([])
      expect(nlogs[1].title).toBe('お茶')
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('「、」で今までどおりブロックから抜けられる', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n、\n次のメモ')
      const requests = await stmt.generate_requests()
      expect(pick_nlog_requests(requests).length).toBe(1)
      expect(requests.filter((request) => request instanceof KFTLKmemoRequest).length).toBe(1)
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    // タグはブロックの中・金額の行のあとに書かせる。前に書かれても黙って捨てない
    test('「ーん」より前のタグはおかしな行になる', async () => {
      const stmt = new KFTLStatement('。買い物\nーん\nコンビニ\nおにぎり\n150')
      expect(await stmt.get_invalid_line_indexs()).toContain(1)
    })

    test('「ーん」より前のテキストブロックもおかしな行になる', async () => {
      const stmt = new KFTLStatement('ーー\nメモ\nーー\nーん\nコンビニ\nおにぎり\n150')
      expect(await stmt.get_invalid_line_indexs()).toContain(3)
    })

    test('店名の位置・最初の品名の位置に書いたタグはおかしな行になる', async () => {
      const shop_position = new KFTLStatement('ーん\n。食費\nおにぎり\n150')
      expect(await shop_position.get_invalid_line_indexs()).toContain(1)
      const title_position = new KFTLStatement('ーん\nコンビニ\n。食費\n150')
      expect(await title_position.get_invalid_line_indexs()).toContain(2)
    })

    test('「ーん」より前の関連時刻はブロックの全支払いに効く', async () => {
      const stmt = new KFTLStatement('？2025-01-15 10:00:00\nーん\nコンビニ\nおにぎり\n150\nお茶\n120')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      expect(nlogs.length).toBe(2)
      for (const nlog of nlogs) {
        const related_time = nlog.get_related_time()!
        expect(related_time.getFullYear()).toBe(2025)
        expect(related_time.getMonth()).toBe(0)
        expect(related_time.getDate()).toBe(15)
      }
    })

    // 関連時刻だけは支払いごとではなくブロック全体。書いた位置より前の支払いにも効く
    test('ブロックの中の関連時刻もブロックの全支払いに効く', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n？2025-01-15 10:00:00\nお茶\n120')
      const stmt_requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(stmt_requests)
      expect(nlogs.length).toBe(2)
      for (const nlog of nlogs) {
        expect(nlog.get_related_time()!.getFullYear()).toBe(2025)
      }
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('小数の金額が切り捨てられずに通る', async () => {
      const stmt = new KFTLStatement('ーん\nカフェ\nコーヒー\n-1.5')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      expect(nlogs[0].amount).toBe(-1.5)
    })

    test('品名だけで金額が無いとエラーになる', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\nお茶')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      const incomplete = nlogs.find((nlog) => nlog.title === 'お茶')!
      const errors = await incomplete.do_request({} as unknown as GkillAPI, {} as unknown as ApplicationConfig)
      expect(errors.length).toBe(1)
      expect(errors[0].error_code).toBe('ERR900014')
    })

    // 末尾の改行が品名行として解釈されるだけなので、エラーにも支払いにもしない
    test('末尾の空行は支払いを作らずエラーにもならない', async () => {
      const stmt = new KFTLStatement('ーん\nコンビニ\nおにぎり\n150\n')
      const nlogs = pick_nlog_requests(await stmt.generate_requests())
      const blank = nlogs.find((nlog) => nlog.title === '')!
      const errors = await blank.do_request({} as unknown as GkillAPI, {} as unknown as ApplicationConfig)
      expect(errors).toEqual([])
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })

    test('ASCII の /expense と # でも同じ結果になる', async () => {
      const stmt = new KFTLStatement('/expense\nConvenience store\nRice ball\n150\n#food\nTea\n120\n#drink')
      const stmt_requests = await stmt.generate_requests()
      const nlogs = pick_nlog_requests(stmt_requests)
      expect(nlogs.length).toBe(2)
      expect(nlogs[0].get_tags()).toEqual(['food'])
      expect(nlogs[1].get_tags()).toEqual(['drink'])
      expect(await stmt.get_invalid_line_indexs()).toEqual([])
    })
  })
})
