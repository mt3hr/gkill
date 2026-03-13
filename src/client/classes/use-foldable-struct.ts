import { computed, ref, watch, type Ref } from 'vue'
import type { FoldableStructEmits } from '@/pages/views/foldable-struct-emits'
import type { FoldableStructProps } from '@/pages/views/foldable-struct-props'
import { CheckState } from '@/pages/views/check-state'
import type { FoldableStructModel } from '@/pages/views/foldable-struct-model'
import { DropTypeFoldableStruct } from '@/classes/api/drop-type-foldable-struct'

export function useFoldableStruct(options: {
    props: FoldableStructProps,
    emits: FoldableStructEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const child_foldable_structs = ref()

    // ── State refs ──
    const open_group: Ref<boolean> = ref(props.is_root ? props.is_open : false)
    const check: Ref<boolean> = ref(false)
    const struct_list: Ref<Array<FoldableStructModel>> = ref(new Array<FoldableStructModel>())
    const indeterminate_group: Ref<boolean> = ref(false)
    const tr_size: Ref<Number> = ref(24)
    const font_size: Ref<Number> = ref(16)

    // ── Computed ──
    const font_size_px = computed(() => font_size.value.valueOf().toString().concat("px"))

    // ── Watchers ──
    watch(() => props.is_open, () => {
        open_group.value = props.is_open
    })
    watch(() => props.struct_obj.is_checked, () => {
        updated_struct()
        update_check()
    })
    watch(() => props.struct_obj.indeterminate, () => {
        updated_struct()
        update_check()
    })
    watch(() => props.struct_obj, () => {
        updated_struct()
        update_check()
    })

    // ── Initialization ──
    open_group.value = props.is_open
    updated_struct()
    update_check()

    // ── Internal helpers ──

    // this.structがアイテムであればtrueを、そうではなくグループである場合はfalseを返します。
    function is_item() {
        return !props.struct_obj.is_dir
    }

    // アイテムではなくの場合に使われます。
    // 子アイテムを子アイテム配列に変換してthis.struct_listに収めます。
    // this.struct_listはv-forで回して子アイテムとして再帰的に読み込まれます。
    function updated_struct() {
        struct_list.value = props.struct_obj.children ? props.struct_obj.children : new Array<FoldableStructModel>()
    }

    // チェックボックスのチェック状態を最新の状態に更新します。
    // 親であれば子の状態を見ます
    function update_check() {
        if (is_item()) {
            check.value = (props.struct_obj).is_checked
        } else {
            if (child_foldable_structs.value) {
                for (let i = 0; i < child_foldable_structs.value.length; i++) {
                    child_foldable_structs.value[i].update_check()
                }
            }
            let exist_checked = false
            let all_checked = true

            let f = (_struct: FoldableStructModel) => { }
            let func = (struct: FoldableStructModel) => {
                if (struct.is_checked) {
                    exist_checked = true
                } else {
                    all_checked = false
                }
                if (struct.children) {
                    struct.children.forEach(child => {
                        f(child)
                    })
                }
            }
            f = func
            struct_list.value.forEach(struct_child => f(struct_child))

            if (all_checked) {
                check.value = true
                indeterminate_group.value = false
            } else if (exist_checked && !all_checked) {
                check.value = false
                indeterminate_group.value = true
            } else {
                indeterminate_group.value = false
                check.value = false
            }
        }
    }

    function get_group_open(_index: number) {
        return false
    }

    // 子アイテムのグループ名を取得するためにv-for内から使われます。
    function get_group_name(index: number) {
        let group_name = struct_list.value[index].key
        if (group_name.endsWith(',close') || group_name.endsWith(', close')) {
            group_name = group_name.split(',').slice(0, -1).join(',')
        } else if (group_name.endsWith(',open') || group_name.endsWith(', open')) {
            group_name = group_name.split(',').slice(0, -1).join(',')
        }
        return group_name
    }

    // アイテムのチェック状態に変更があった場合に呼び出されます。
    // すべての子アイテムのcheckの状態を、グループのチェック状態と同じにします。
    function change_group_by_user() {
        let items = new Array()
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            items.push(struct.key)
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        f(props.struct_obj)
        emit_updated_check_items_by_user(items, check.value, false)
    }

    // 子グループ内のアイテムに変更があったときに、それを上に伝えるために呼び出されます。
    function emit_updated_check_items_by_user(items: Array<string>, check: boolean, indeterminate: boolean) {
        emits('requested_update_check_state', items, indeterminate ? CheckState.indeterminate : check ? CheckState.checked : CheckState.unchecked)
    }

    // 子グループ内の複数のアイテムのみをチェックするように変更があったときに、それを上に伝えるために呼び出されます。
    function emit_click_items_by_user(e: MouseEvent, items: Array<string>) {
        emits('clicked_items', e, items, CheckState.checked, true)
    }

    // 子グループ内の一つのアイテムのみをチェックするよう変更があったときに、それを上に伝えるために呼び出されます。
    function emit_click_item_by_user(e: MouseEvent, item: string) {
        emit_click_items_by_user(e, [item])
    }

    // アイテムのチェック状態に変更があったときに呼び出されます。
    function update_check_item_by_user() {
        emit_updated_check_items_by_user([(props.struct_obj).key], check.value, (props.struct_obj).indeterminate)
    }

    // このアイテムがクリックされたときに呼び出されます。
    // このアイテムのみにチェックが入るように上にemitします。
    function click_item_by_user(e: MouseEvent) {
        emit_click_item_by_user(e, props.struct_obj.key)
    }

    function dblclick_item_by_user(e: MouseEvent) {
        emits('dblclicked_item', e, props.struct_obj.id)
    }

    // このアイテムがクリックされたときに呼び出されます。
    // このアイテム内のアイテムのみにチェックが入るように上にemitします。
    function click_group_by_user(e: MouseEvent) {
        let items = new Array<string>()
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            items.push(struct.key)
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        f(props.struct_obj)
        emit_click_items_by_user(e, items)
    }

    // 現在チェックの入っているアイテム名を配列で取得します。
    function get_selected_items(): Array<string> {
        let items = new Array<string>()
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            if (struct.is_checked) {
                items.push(struct.key)
            }
            if (struct.children) {
                struct.children.forEach(child => {
                    f(child)
                })
            }
        }
        f = func
        f(props.struct_obj)
        return items
    }

    function drag_start(e: DragEvent): void {
        // 編集が有効でなければ何もしない
        if (!props.is_editable) {
            return
        }

        // struct_objをJSONにしてdataTransferにセット
        e.dataTransfer?.setData("gkill_struct_obj_json", JSON.stringify(props.struct_obj))
        e.stopPropagation()
    }

    function drop(e: DragEvent): void {
        // 編集が有効でなければ何もしない
        if (!props.is_editable) {
            return
        }

        // struct_objをJSONから復元
        const struct_obj_json = e.dataTransfer?.getData("gkill_struct_obj_json")
        if (!struct_obj_json) {
            return
        }
        const struct_obj: FoldableStructModel = JSON.parse(struct_obj_json) as unknown as FoldableStructModel


        // 自分自身にドロップしていたら何もしない
        if (struct_obj.id === props.struct_obj.id) {
            return
        }

        // ドロップされたものを移動。
        // 移動する場所の決定
        let drop_type: DropTypeFoldableStruct = DropTypeFoldableStruct.up_element
        if (props.struct_obj.children) { // フォルダの場合
            if (e.offsetY <= tr_size.value.valueOf() * (1 / 3)) {
                drop_type = DropTypeFoldableStruct.up_element
            } else if (e.offsetY <= tr_size.value.valueOf() * (2 / 3)) {
                drop_type = DropTypeFoldableStruct.in_folder_bottom
            } else {
                drop_type = DropTypeFoldableStruct.down_element
            }
        } else { // フォルダではない要素の場合
            if (e.offsetY <= tr_size.value.valueOf() * (1 / 2)) {
                drop_type = DropTypeFoldableStruct.up_element
            } else {
                drop_type = DropTypeFoldableStruct.down_element
            }
        }
        emits('requested_move_struct_obj', struct_obj, props.struct_obj, drop_type)
        // e.preventDefault()
        e.stopPropagation()
    }

    // ドロップされたときの移動処理
    function handle_move_struct_obj(struct_obj: FoldableStructModel, target_struct_obj: FoldableStructModel, drop_type: DropTypeFoldableStruct): void {
        // rootのみで再帰的にやる
        if (!props.is_root) {
            emits('requested_move_struct_obj', struct_obj, target_struct_obj, drop_type)
            return
        }

        // parentのなかにchildがあったらtrueを返す
        let has_child = (_parent: FoldableStructModel, _child: FoldableStructModel): boolean => false
        let has_child_impl = (parent: FoldableStructModel, child: FoldableStructModel): boolean => {
            let is_has_child = false
            if (parent.children) {
                for (let i = 0; i < parent.children.length; i++) {
                    if (parent.children[i].id === child.id) {
                        is_has_child = true
                        break
                    }
                    if (has_child(parent.children[i], child)) {
                        is_has_child = true
                        break
                    }
                }
            }
            return is_has_child
        }
        has_child = has_child_impl

        // 親を子に入れようとしていたらおかしくなるので何もしない
        if (has_child(struct_obj, target_struct_obj)) {
            return
        }

        // 再帰的にやる
        if (!struct_obj.id) {
            return
        }
        delete_struct(struct_obj.id)
        let pasted = false
        let f = (_walk_struct_obj: FoldableStructModel, _parent_struct_obj: FoldableStructModel) => { }
        let func = (walk_struct_obj: FoldableStructModel, parent_struct_obj: FoldableStructModel) => {
            if (pasted) {
                return
            }

            switch (drop_type) {
                case DropTypeFoldableStruct.in_folder_top:
                    // 自分が対象であればいれる フォルダの場合
                    if (walk_struct_obj.id !== target_struct_obj.id) {
                        break
                    }
                    if (!walk_struct_obj.children) {
                        break
                    }
                    walk_struct_obj.children.unshift(struct_obj)
                    pasted = true
                    break
                case DropTypeFoldableStruct.in_folder_bottom:
                    // 自分が対象であればいれる フォルダの場合
                    if (walk_struct_obj.id !== target_struct_obj.id) {
                        break
                    }
                    if (!walk_struct_obj.children) {
                        break
                    }
                    walk_struct_obj.children.push(struct_obj)
                    pasted = true
                    break
                case DropTypeFoldableStruct.up_element:
                    // 自分が対象であればいれる フォルダではない場合
                    if (!parent_struct_obj.children) {
                        break
                    }
                    for (let i = 0; i < parent_struct_obj.children.length; i++) {
                        if (target_struct_obj.id === parent_struct_obj.children[i].id) {
                            parent_struct_obj.children.splice(i, 0, struct_obj)
                            pasted = true
                            break
                        }
                    }
                    break
                case DropTypeFoldableStruct.down_element:
                    // 自分が対象であればいれる フォルダではない場合
                    if (!parent_struct_obj.children) {
                        break
                    }
                    for (let i = 0; i < parent_struct_obj.children.length; i++) {
                        if (target_struct_obj.id === parent_struct_obj.children[i].id) {
                            parent_struct_obj.children.splice(i + 1, 0, struct_obj)
                            pasted = true
                            break
                        }
                    }
                    break
            }

            if (!pasted) {
                if (walk_struct_obj.children) {
                    for (let i = 0; i < walk_struct_obj.children.length; i++) {
                        f(walk_struct_obj.children[i], walk_struct_obj)
                    }
                }
            }
        }
        f = func
        struct_list.value.forEach(struct_child => f(struct_child, props.struct_obj))
    }

    function get_foldable_struct(): Array<FoldableStructModel> {
        return struct_list.value.concat()
    }

    function dragover(e: DragEvent): void {
        if (e.dataTransfer) {
            e.dataTransfer.dropEffect = "move"
        }
        e.preventDefault()
        e.stopPropagation()
    }

    function update_struct_obj(struct_obj: FoldableStructModel): void {
        if (!props.is_root) {
            emits('requested_update_struct_obj', struct_obj)
            return
        }
        let f = (_struct: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel) => {
            if (!struct.children) {
                return
            }
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i])
                if (struct_obj.id === struct.children[i].id) {
                    struct.children.splice(i, 1, struct_obj)
                    break
                }
            }
        }
        f = func
        for (let i = 0; i < struct_list.value.length; i++) {
            f(struct_list.value[i])
        }
    }

    function delete_struct(id: string): boolean {
        if (!props.is_root) {
            return false
        }
        let deleted = false
        let f = (_struct: FoldableStructModel, _parent: FoldableStructModel) => { }
        let func = (struct: FoldableStructModel, parent: FoldableStructModel) => {
            if (deleted) {
                return
            }

            // 子に渡されたものがあれば消す
            if (parent.children) {
                for (let i = 0; i < parent.children.length; i++) {
                    if (parent.children[i].id === id) {
                        parent.children.splice(i, 1)
                        emits('requested_update_struct_obj', struct)
                        deleted = true
                        break
                    }
                }
            }
            if (struct.children) {
                for (let i = 0; i < struct.children.length; i++) {
                    f(struct.children[i], struct)
                }
            }
        }
        f = func
        struct_list.value.forEach(struct_child => f(struct_child, props.struct_obj))
        return deleted
    }

    // ── Template event handlers ──
    function onContextmenuItem(e: MouseEvent) {
        emits('contextmenu_item', e, props.struct_obj.id)
    }

    function onToggleOpenGroup() {
        open_group.value = !open_group.value
    }

    function onChildReceivedErrors(...errors: any[]) {
        emits('received_errors', errors[0] as Array<import('@/classes/api/gkill-error').GkillError>)
    }

    function onChildReceivedMessages(...messages: any[]) {
        emits('received_messages', messages[0] as Array<import('@/classes/api/gkill-message').GkillMessage>)
    }

    function onChildDblclickedItem(e: MouseEvent, id: string | null) {
        emits('dblclicked_item', e, id)
    }

    function onChildContextmenuItem(e: MouseEvent, id: string | null) {
        emits('contextmenu_item', e, id)
    }

    function onChildRequestedUpdateCheckState(items: Array<string>, check_state: CheckState) {
        emits('requested_update_check_state', items, check_state)
    }

    // ── Return ──
    return {
        // Template refs
        child_foldable_structs,

        // State
        open_group,
        check,
        struct_list,
        indeterminate_group,
        font_size_px,

        // Methods used in template
        is_item,
        get_group_open,
        get_group_name,
        drag_start,
        drop,
        dragover,
        update_check_item_by_user,
        click_item_by_user,
        dblclick_item_by_user,
        change_group_by_user,
        click_group_by_user,
        emit_click_items_by_user,
        update_struct_obj,

        // Template event handlers
        onContextmenuItem,
        onToggleOpenGroup,
        onChildReceivedErrors,
        onChildReceivedMessages,
        onChildDblclickedItem,
        onChildContextmenuItem,
        onChildRequestedUpdateCheckState,

        // Exposed methods
        get_selected_items,
        handle_move_struct_obj,
        get_foldable_struct,
        delete_struct,
        update_check,
    }
}
