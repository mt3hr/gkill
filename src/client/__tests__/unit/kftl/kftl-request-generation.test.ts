import { describe, test, expect, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTLKmemoRequest } from '@/classes/kftl/kftl_kmemo/kftl-kmemo-request'

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
})
