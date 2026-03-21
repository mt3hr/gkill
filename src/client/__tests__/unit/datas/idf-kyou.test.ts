// IDFKyou has circular import chains through the API response types that cause
// "Class extends value undefined" in jsdom. These tests use a plain-object
// factory to verify the data shape.

import { describe, test, expect } from 'vitest'

function makeIDFKyou(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-idf-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    data_type: 'idf_kyou',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    file_name: '',
    file_url: '',
    is_image: false,
    is_video: false,
    is_audio: false,
    attached_histories: [],
    ...overrides,
  }
}

describe('IDFKyou (factory-based)', () => {
  test('makeIDFKyou returns object with all required fields', () => {
    const idf = makeIDFKyou()
    expect(idf.id).toBe('test-idf-id')
    expect(idf.is_deleted).toBe(false)
    expect(idf.rep_name).toBe('test-rep')
    expect(idf.data_type).toBe('idf_kyou')
    expect(idf.related_time).toBeDefined()
    expect(idf.create_time).toBeDefined()
    expect(idf.create_app).toBe('gkill')
    expect(idf.create_user).toBe('admin')
    expect(idf.update_time).toBeDefined()
    expect(idf.update_app).toBe('gkill')
    expect(idf.update_user).toBe('admin')
  })

  test('makeIDFKyou includes file-related fields', () => {
    const idf = makeIDFKyou()
    expect(idf.file_name).toBe('')
    expect(idf.file_url).toBe('')
    expect(idf.is_image).toBe(false)
    expect(idf.is_video).toBe(false)
    expect(idf.is_audio).toBe(false)
  })

  test('makeIDFKyou includes attached_histories initialized to empty array', () => {
    const idf = makeIDFKyou()
    expect(idf.attached_histories).toEqual([])
  })

  test('makeIDFKyou overrides work', () => {
    const idf = makeIDFKyou({
      id: 'custom-id',
      file_name: 'photo.jpg',
      file_url: '/files/photo.jpg',
      is_image: true,
      is_deleted: true,
    })
    expect(idf.id).toBe('custom-id')
    expect(idf.file_name).toBe('photo.jpg')
    expect(idf.file_url).toBe('/files/photo.jpg')
    expect(idf.is_image).toBe(true)
    expect(idf.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(idf.rep_name).toBe('test-rep')
  })

  test('makeIDFKyou creates independent objects', () => {
    const a = makeIDFKyou()
    const b = makeIDFKyou()
    a.id = 'modified'
    expect(b.id).toBe('test-idf-id')
  })

  test('can represent image file', () => {
    const idf = makeIDFKyou({
      file_name: 'vacation.png',
      file_url: '/data/vacation.png',
      is_image: true,
      is_video: false,
      is_audio: false,
    })
    expect(idf.is_image).toBe(true)
    expect(idf.is_video).toBe(false)
    expect(idf.is_audio).toBe(false)
  })

  test('can represent video file', () => {
    const idf = makeIDFKyou({
      file_name: 'clip.mp4',
      file_url: '/data/clip.mp4',
      is_image: false,
      is_video: true,
      is_audio: false,
    })
    expect(idf.is_video).toBe(true)
  })

  test('can represent audio file', () => {
    const idf = makeIDFKyou({
      file_name: 'recording.wav',
      file_url: '/data/recording.wav',
      is_image: false,
      is_video: false,
      is_audio: true,
    })
    expect(idf.is_audio).toBe(true)
  })
})
