package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

// Message paths (must match watch_app GkillWearClient)
internal const val PATH_GET_TEMPLATES = "/gkill/get_templates"
internal const val PATH_TEMPLATES = "/gkill/templates"
internal const val PATH_SUBMIT = "/gkill/submit"
internal const val PATH_SUBMIT_FORCE = "/gkill/submit_force"
internal const val PATH_SUBMIT_RESULT = "/gkill/submit_result"
internal const val PATH_GET_PLAING_TIMEIS = "/gkill/get_plaing_timeis"
internal const val PATH_PLAING_TIMEIS = "/gkill/plaing_timeis"
internal const val PATH_END_TIMEIS = "/gkill/end_timeis"
internal const val PATH_END_TIMEIS_RESULT = "/gkill/end_timeis_result"

/** ワーカーがディスパッチするリクエストパス（時計→スマホ）。 */
internal val KNOWN_REQUEST_PATHS = setOf(
    PATH_GET_TEMPLATES,
    PATH_SUBMIT,
    PATH_SUBMIT_FORCE,
    PATH_GET_PLAING_TIMEIS,
    PATH_END_TIMEIS,
)

/**
 * 時計から届いた1件の要求を gkill サーバーAPIへ変換し、返すべき応答（パスとバイト列）を組み立てる。
 *
 * Android フレームワークに依存しない素のクラスにしてあるので、[GkillApiClient] を MockWebServer で
 * 差し替えれば JVM 単体テストできる（[GkillWearableListenerService] / [WearRequestWorker] は
 * 破棄やプロセス死を跨ぐが、ここは純粋なロジックに保つ）。
 *
 * @param sessionProvider 有効なセッションIDを返す。失敗時は null（→ `ERROR:login_failed`）。
 *   実運用ではキャッシュセッションの検証と再ログインを行うワーカー側の関数を渡す。
 * @param submitLedger 直近成功した KFTL テキストの台帳。重複送信を検出する。null なら無効。
 * @param idempotencyKey KFTL 送信に付けるサーバー側冪等キー。ワーカー再送で同じ値を送ると
 *   サーバーが1回の登録に畳む（結果だけ届かなかった場合の二重登録を防ぐ）。null なら付けない。
 */
class WearRequestHandler(
    private val apiClient: GkillApiClient,
    private val sessionProvider: () -> String?,
    private val submitLedger: WearSubmitLedger? = null,
    private val idempotencyKey: String? = null,
) {

    /** 応答。[path] へ [data] を送り返す。 */
    class Response(val path: String, val data: ByteArray)

    /** 未知パスなら null（応答不要）。 */
    fun handle(path: String, requestData: ByteArray): Response? = when (path) {
        PATH_GET_TEMPLATES -> handleGetTemplates()
        PATH_SUBMIT -> handleSubmit(requestData, force = false)
        PATH_SUBMIT_FORCE -> handleSubmit(requestData, force = true)
        PATH_GET_PLAING_TIMEIS -> handleGetPlaingTimeis()
        PATH_END_TIMEIS -> handleEndTimeis(requestData)
        else -> null
    }

    fun handleGetTemplates(): Response {
        val session = sessionProvider()
            ?: return resp(PATH_TEMPLATES, "ERROR:login_failed")
        val templatesJson = apiClient.getKftlTemplateStructJson(session)
            ?: return resp(PATH_TEMPLATES, "ERROR:get_config_failed")
        return resp(PATH_TEMPLATES, templatesJson)
    }

    /**
     * KFTL テキストをサーバーへ送る。
     *
     * 重複対策は share-target-dedup の契約を写したもの:
     * - 台帳へ載せるのは保存が成功したときだけ（応答を見てから記録）。
     * - 再配送と意図的な再送は内容から区別できないので、黙って捨てず `DUPLICATE` を返して
     *   時計側に確認（それでも送信）を出させる。[force]=true はその確認経由の明示的な再送。
     */
    fun handleSubmit(requestData: ByteArray, force: Boolean): Response {
        val kftlText = String(requestData, Charsets.UTF_8)
        if (!force && submitLedger?.isDuplicate(kftlText) == true) {
            return resp(PATH_SUBMIT_RESULT, "DUPLICATE")
        }
        val session = sessionProvider()
            ?: return resp(PATH_SUBMIT_RESULT, "ERROR:login_failed")
        val error = apiClient.submitKFTLText(session, kftlText, idempotencyKey)
        return if (error == null) {
            submitLedger?.recordSuccess(kftlText)
            resp(PATH_SUBMIT_RESULT, "OK")
        } else {
            resp(PATH_SUBMIT_RESULT, "ERROR:$error")
        }
    }

    fun handleGetPlaingTimeis(): Response {
        val session = sessionProvider()
            ?: return resp(PATH_PLAING_TIMEIS, "ERROR:login_failed")
        val result = apiClient.getPlaingTimeis(session)
            ?: return resp(PATH_PLAING_TIMEIS, "ERROR:get_plaing_timeis_failed")
        return resp(PATH_PLAING_TIMEIS, result)
    }

    fun handleEndTimeis(requestData: ByteArray): Response {
        val data = String(requestData, Charsets.UTF_8)
        // data format: "id\nrep_name"
        val parts = data.split("\n", limit = 2)
        val timeisId = parts.getOrNull(0) ?: ""
        val repName = parts.getOrNull(1) ?: ""
        if (timeisId.isEmpty()) {
            return resp(PATH_END_TIMEIS_RESULT, "ERROR:empty_timeis_id")
        }
        val session = sessionProvider()
            ?: return resp(PATH_END_TIMEIS_RESULT, "ERROR:login_failed")
        val error = apiClient.endTimeis(session, timeisId, repName)
        return if (error == null) {
            resp(PATH_END_TIMEIS_RESULT, "OK")
        } else {
            resp(PATH_END_TIMEIS_RESULT, "ERROR:$error")
        }
    }

    private fun resp(path: String, text: String): Response =
        Response(path, text.toByteArray(Charsets.UTF_8))
}
