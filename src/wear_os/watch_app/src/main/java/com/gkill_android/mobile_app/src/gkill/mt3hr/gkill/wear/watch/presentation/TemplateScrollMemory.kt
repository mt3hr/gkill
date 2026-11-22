package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation

/** テンプレート一覧のスクロール位置。`ScalingLazyListState` の中央項目で表す。 */
data class TemplateScrollPosition(val centerItemIndex: Int, val centerItemScrollOffset: Int) {
    companion object {
        /**
         * 先頭から見せる位置。index 1 は `ScalingLazyColumn` の既定で、
         * item 0 のタイトル行の次（＝最初のチップ）が中央に来る。
         */
        val TOP = TemplateScrollPosition(1, 0)
    }
}

/**
 * テンプレート一覧の、階層ごとのスクロール位置の記憶。
 *
 * - 親→子（フォルダを開く）: 親の位置を積み、子は先頭から見せる
 * - 子→親（Back で戻る）: 積んだ位置を1回だけ復元する
 * - それ以外（読み込み直し・確認画面からの復帰・ホームへ戻る）: 先頭から
 *
 * `MainActivity` の `is Screen.TemplateList ->` は1つの呼び出し位置なので、`ScalingLazyColumn` に
 * state を渡さないと既定の `rememberScalingLazyListState()` がそこに remember され、
 * どの移動でもエラーも警告も出ないまま前の階層のスクロール位置を引き継ぐ。
 * 移動の意味（掘ったのか戻ったのか）を知っているのは画面遷移を握る `MainActivity` だけなので、
 * 方針はここに持たせ、新しく作る state の初期値として渡す。
 */
class TemplateScrollMemory {

    /** 掘るたびに積まれる、親の階層の位置。深さぶんだけ溜まる。 */
    private val stack = ArrayDeque<TemplateScrollPosition>()

    /** 次に作る階層で1回だけ使う復元位置。子→親のときだけ入る。 */
    private var pending: TemplateScrollPosition? = null

    /** 親→子。いま見ている親の位置を積む（子は先頭から見せる）。 */
    fun enterFolder(current: TemplateScrollPosition) {
        stack.addLast(current)
        pending = null
    }

    /** 子→親。親で復元する位置を用意する（積んだものが無ければ先頭のまま）。 */
    fun backToParent() {
        pending = stack.removeLastOrNull()
    }

    /** 一覧を作り直すとき（読み込み直し・ホームへ戻る）。積んだ位置を全部忘れる。 */
    fun reset() {
        stack.clear()
        pending = null
    }

    /** 新しく作る階層の初期位置。復元は1回きりで、以後は先頭から。 */
    fun takeInitialPosition(): TemplateScrollPosition {
        val restored = pending
        pending = null
        return restored ?: TemplateScrollPosition.TOP
    }
}
