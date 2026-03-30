/**
 * save-as.ts tests.
 */
import { vi } from 'vitest'

// jsdom doesn't have URL.createObjectURL/revokeObjectURL
if (typeof URL.createObjectURL === 'undefined') {
  (URL as unknown as Record<string, unknown>).createObjectURL = () => 'blob:mock-url'
}
if (typeof URL.revokeObjectURL === 'undefined') {
  (URL as unknown as Record<string, unknown>).revokeObjectURL = () => {}
}

import { saveAs } from '@/classes/save-as'

describe('saveAs', () => {
  let mockClick: ReturnType<typeof vi.fn>
  let mockAnchor: Partial<HTMLAnchorElement>

  beforeEach(() => {
    mockClick = vi.fn()
    mockAnchor = {
      style: {} as CSSStyleDeclaration,
      href: '',
      download: '',
      click: mockClick,
    }
    vi.spyOn(document, 'createElement').mockReturnValue(mockAnchor as HTMLAnchorElement)
    vi.spyOn(document.body, 'appendChild').mockImplementation((node) => node)
    vi.spyOn(document.body, 'removeChild').mockImplementation((node) => node)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test-url')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('creates and clicks a download anchor element', () => {
    saveAs(new Blob(['test']), 'test.txt')
    expect(document.createElement).toHaveBeenCalledWith('a')
    expect(mockClick).toHaveBeenCalled()
    expect(mockAnchor.download).toBe('test.txt')
  })

  test('converts string to Blob with octet-stream type', () => {
    saveAs('hello world', 'file.txt')
    const createObjCall = (URL.createObjectURL as ReturnType<typeof vi.fn>).mock.calls[0][0]
    expect(createObjCall).toBeInstanceOf(Blob)
  })

  test('passes through Blob directly', () => {
    const blob = new Blob(['data'], { type: 'text/plain' })
    saveAs(blob, 'data.txt')
    expect(URL.createObjectURL).toHaveBeenCalledWith(blob)
  })

  test('revokes object URL after click', () => {
    saveAs('test', 'file.txt')
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test-url')
    expect(document.body.removeChild).toHaveBeenCalled()
  })
})
