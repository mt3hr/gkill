/**
 * useRykvDetailPaneMetrics（rykv の詳細ペインと Ryuu の寸法）。
 *
 * ポート（rudbeckia）のウィンドウの中でも描かれるので、基準は画面ではなく渡された箱の寸法（100vw を使わない）。
 */
import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useRykvDetailPaneMetrics } from '@/classes/use-rykv-detail-pane-metrics'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

describe('useRykvDetailPaneMetrics', () => {
    it('Ryuu 定義が空のときは詳細ペインに高さ100%を配り、定義があるときは20%を Ryuu に配る', () => {
        const config = ref({ ryuu_json_data: [] } as unknown as ApplicationConfig)
        const height = ref(1000)
        const width = ref(800)

        const metrics = useRykvDetailPaneMetrics({
            application_config: () => config.value,
            app_content_height: () => height.value,
            app_content_width: () => width.value,
        })

        expect(metrics.is_ryuu_empty.value).toBe(true)
        expect(metrics.ryuu_pane_height_px.value).toBe('0px')
        expect(metrics.kyou_detail_view_height_px.value).toBe('1000px')
        expect(metrics.kyou_detail_view_base_width_px.value).toBe('400px')

        config.value = { ryuu_json_data: [{ queries: [{ id: 'q1' }] }] } as unknown as ApplicationConfig
        expect(metrics.is_ryuu_empty.value).toBe(false)
        expect(metrics.ryuu_pane_height_px.value).toBe('200px')
        expect(metrics.kyou_detail_view_height_px.value).toBe('800px')

        // 幅が 600px 以下のときは箱の幅いっぱいを使う（100vw を使わない）
        width.value = 500
        expect(metrics.kyou_detail_view_base_width_px.value).toBe('500px')
    })
})
