import { GitCommitLog } from '@/classes/datas/git-commit-log'

describe('GitCommitLog', () => {
  test('can be instantiated', () => {
    const log = new GitCommitLog()
    expect(log).toBeInstanceOf(GitCommitLog)
  })

  describe('default field values', () => {
    let log: GitCommitLog

    beforeEach(() => {
      log = new GitCommitLog()
    })

    test('commit_message defaults to empty string', () => {
      expect(log.commit_message).toBe('')
    })

    test('addition defaults to 0', () => {
      expect(log.addition).toBe(0)
    })

    test('deletion defaults to 0', () => {
      expect(log.deletion).toBe(0)
    })

    test('attached_histories defaults to empty array', () => {
      expect(log.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(log.is_deleted).toBe(false)
      expect(log.id).toBe('')
      expect(log.rep_name).toBe('')
      expect(log.data_type).toBe('')
      expect(log.create_app).toBe('')
      expect(log.create_device).toBe('')
      expect(log.create_user).toBe('')
      expect(log.update_app).toBe('')
      expect(log.update_user).toBe('')
      expect(log.update_device).toBe('')
      expect(log.related_time).toBeInstanceOf(Date)
      expect(log.create_time).toBeInstanceOf(Date)
      expect(log.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const log = new GitCommitLog()
      log.id = 'commit-001'
      log.commit_message = 'fix: resolve bug'
      log.addition = 10
      log.deletion = 3
      log.rep_name = 'my-repo'
      log.is_deleted = true

      const cloned = log.clone()

      expect(cloned).toBeInstanceOf(GitCommitLog)
      expect(cloned.id).toBe('commit-001')
      expect(cloned.commit_message).toBe('fix: resolve bug')
      expect(cloned.addition).toBe(10)
      expect(cloned.deletion).toBe(3)
      expect(cloned.rep_name).toBe('my-repo')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const log = new GitCommitLog()
      const cloned = log.clone()
      expect(cloned).not.toBe(log)
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const log = new GitCommitLog()
      const errors = await log.clear_attached_histories()
      expect(errors).toEqual([])
      expect(log.attached_histories).toEqual([])
    })
  })
})
