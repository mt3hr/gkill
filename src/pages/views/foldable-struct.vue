<template>
    <tr v-if="is_item()" :draggable="is_editable" @dragstart="drag_start" @drop="drop" :dropzone="is_editable"
        :key="props.struct_obj.key" @dragover="dragover"
        @contextmenu.prevent.stop="(e: MouseEvent) => emits('contextmenu_item', e, props.struct_obj.id)">
        <td>
            <table>
                <tr>
                    <td v-if="is_show_checkbox">
                        <input type="checkbox" class="checkbox_in_foldable_struct" v-model="check"
                            @change="update_check_item_by_user" :indeterminate.prop="(struct_obj).indeterminate" />
                    </td>
                    <td class="tree_item ml-1" @dblclick="dblclick_item_by_user"
                        @click.prevent.stop="click_item_by_user">{{ struct_obj.name }}</td>
                </tr>
            </table>
        </td>
    </tr>
    <tr v-if="!is_item()" :draggable="is_editable" @dragstart="drag_start" @drop="drop" :dropzone="is_editable"
        :key="props.struct_obj.key" @dragover="dragover"
        @contextmenu.prevent.stop="(e: MouseEvent) => emits('contextmenu_item', e, props.struct_obj.id)">
        <td>
            <table>
                <tr>
                    <td v-if="is_show_checkbox">
                        <input type="checkbox" class="checkbox_in_foldable_struct" v-model="check"
                            @change="change_group_by_user" :indeterminate="indeterminate_group" />
                    </td>
                    <td>
                        <span v-if="open_group" style="cursor: default" @click="open_group = !open_group">▽</span>
                        <span v-if="!open_group" style="cursor: default" @click="open_group = !open_group">▷</span>
                    </td>
                    <td @click="click_group_by_user">
                        <div class="tree_item">{{ folder_name }}</div>
                    </td>
                </tr>
            </table>
            <table class="ml-4" v-if="open_group">
                <FoldableStruct v-show="open_group" v-for="child_struct, index in struct_list"
                    :key="child_struct.id ? child_struct.id : ''" :application_config="application_config"
                    :folder_name="get_group_name(index)" :gkill_api="gkill_api" :is_open="get_group_open(index)"
                    :struct_obj="child_struct" :is_editable="is_editable" @clicked_items="emit_click_items_by_user"
                    :is_show_checkbox="is_show_checkbox" @click_items_by_user="emit_click_items_by_user"
                    :is_root="false" @received_errors="(errors) => emits('received_errors', errors)"
                    @received_messages="(messages) => emits('received_messages', messages)"
                    @dblclicked_item="(e: MouseEvent, id: string | null) => emits('dblclicked_item', e, id)"
                    @contextmenu_item.prevent.stop="(e: MouseEvent, id: string | null) => emits('contextmenu_item', e, id)"
                    @requested_update_check_state="(items: Array<string>, check_state: CheckState) => emits('requested_update_check_state', items, check_state)"
                    @requested_move_struct_obj="handle_move_struct_obj" @requested_update_struct_obj="update_struct_obj"
                    ref="child_foldable_structs" />
            </table>
        </td>
    </tr>
</template>
<script lang="ts" setup>
import FoldableStruct from './foldable-struct.vue'
import { computed, ref, watch, type Ref } from 'vue'
import type { FoldableStructEmits } from './foldable-struct-emits'
import type { FoldableStructProps } from './foldable-struct-props'
import { CheckState } from './check-state'
import type { FoldableStructModel } from './foldable-struct-model'
import { DropTypeFoldableStruct } from '@/classes/api/drop-type-foldable-struct'

const child_foldable_structs = ref()

const props = defineProps<FoldableStructProps>()
const emits = defineEmits<FoldableStructEmits>()
defineExpose({ get_selected_items, handle_move_struct_obj, get_foldable_struct, delete_struct, update_check })

const open_group: Ref<boolean> = ref(props.is_root ? props.is_open : false)
const check: Ref<boolean> = ref(false)
const struct_list: Ref<Array<FoldableStructModel>> = ref(new Array<FoldableStructModel>())
const indeterminate_group: Ref<boolean> = ref(false)
const tr_size: Ref<Number> = ref(24)
const font_size: Ref<Number> = ref(16)
const font_size_px = computed(() => font_size.value.valueOf().toString().concat("px"))

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

open_group.value = props.is_open
updated_struct()
update_check()

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
// アイテムではなくの場合に使われます。
// 子アイテムを子アイテム配列に変換してthis.struct_listに収めます。
// this.struct_listはv-forで回して子アイテムとして再帰的に読み込まれます。
function updated_struct() {
    struct_list.value = props.struct_obj.children ? props.struct_obj.children : new Array<FoldableStructModel>()
}
// this.structがアイテムであればtrueを、そうではなくグループである場合はfalseを返します。
function is_item() {
    return props.struct_obj.children === null
}
function get_group_open(_index: number) {
    return false
    /*
    let group_name = struct_list.value[index].key
    if (group_name.endsWith(',close') || group_name.endsWith(', close')) {
        return false
    } else if (group_name.endsWith(',open') || group_name.endsWith(', open')) {
        return true
    }
    return true
    */
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
</script>

<style>
.tree_item {
    min-width: 200px;
    cursor: default;
    font-size: v-bind(font_size_px);
}

.checkbox_in_foldable_struct {
    accent-color: rgb(var(--v-theme-primary));
}
</style>