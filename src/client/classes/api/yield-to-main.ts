'use strict'

interface SchedulerLike {
    yield?: () => Promise<void>
}

/**
 * 長い同期ループの合間にメインスレッドへ制御を返す。
 *
 * `scheduler.yield` があればそれを使う(継続が優先スケジュールされ、
 * setTimeoutのネストクランプも受けない)。無い環境
 * (デスクトップ版のElectron 22 = Chromium 108、vitestのjsdom)では
 * `setTimeout(0)` に落とす。
 * 判定を呼び出しごとに行うのはテストでschedulerを差し替えられるようにするため
 * (チャンク単位でしか呼ばれないので実行コストは誤差)。
 */
export function yield_to_main(): Promise<void> {
    const scheduler = (globalThis as { scheduler?: SchedulerLike }).scheduler
    if (scheduler && typeof scheduler.yield === 'function') {
        return scheduler.yield()
    }
    return new Promise<void>(resolve => setTimeout(resolve, 0))
}
