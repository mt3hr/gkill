<template>
    <tr v-if="is_item()">
        <td>
            <table>
                <tr>
                    <td>
                        <input type="checkbox" v-model="check" @change="update_check_item_by_user"
                            :indeterminate.prop="(struct_obj).indeterminate" />
                    </td>
                    <td class="tree_item ml-1" @click.prevent.stop="click_item_by_user">{{ (struct_obj).key }}</td>
                </tr>
            </table>
        </td>
    </tr>
    <tr v-else>
        <td>
            <table>
                <tr>
                    <td>
                        <input type="checkbox" v-model="check" @change="change_group_by_user"
                            :indeterminate="indeterminate_group" />
                    </td>
                    <td>
                        <span v-if="open_group" style="cursor: default" @click="open_group = !open_group">▽</span>
                        <span v-else style="cursor: default" @click="open_group = !open_group">▷</span>
                    </td>
                    <td @click="click_group_by_user">
                        <div class="tree_item">{{ folder_name }}</div>
                    </td>
                </tr>
            </table>
            <table class="ml-4">
                <FoldableStruct v-show="open_group" v-for="child_struct, index in struct_list"
                    :application_config="application_config" :folder_name="get_group_name(index)" :gkill_api="gkill_api"
                    :is_open="get_group_open(index)" :query="query" :struct_obj="child_struct"
                    @clicked_items="emit_click_items_by_user" :key="index"
                    @click_items_by_user="emit_click_items_by_user"
                    @received_errors="(errors) => emits('received_errors', errors)"
                    @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct" />
            </table>
        </td>
    </tr>
</template>
<script lang="ts" setup>
import FoldableStruct from './foldable-struct.vue'
import { ref, watch, type Ref } from 'vue'
import type { FoldableStructEmits } from './foldable-struct-emits'
import type { FoldableStructProps } from './foldable-struct-props'
import { CheckState } from './check-state'
import type { FoldableStructModel } from './foldable-struct-model'

const props = defineProps<FoldableStructProps>()
const emits = defineEmits<FoldableStructEmits>()
defineExpose({ get_selected_items })

const open_group: Ref<boolean> = ref(props.is_open)
const check: Ref<boolean> = ref(false)
const struct_list: Ref<Array<FoldableStructModel>> = ref(new Array<FoldableStructModel>())
const indeterminate_group: Ref<boolean> = ref(false)

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
        let exist_checked = false
        let all_checked = true

        let f = (struct: FoldableStructModel) => { }
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
function get_group_open(index: number) {
    let group_name = Object.keys(props.struct_obj)[index]
    if (group_name.endsWith(',close') || group_name.endsWith(', close')) {
        return false
    } else if (group_name.endsWith(',open') || group_name.endsWith(', open')) {
        return true
    }
    return true
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
    let f = (struct: FoldableStructModel) => { }
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
function emit_click_items_by_user(items: Array<string>) {
    emits('clicked_items', items, CheckState.checked, true)
}
// 子グループ内の一つのアイテムのみをチェックするよう変更があったときに、それを上に伝えるために呼び出されます。
function emit_click_item_by_user(item: string) {
    emit_click_items_by_user([item])
}
// アイテムのチェック状態に変更があったときに呼び出されます。
function update_check_item_by_user() {
    emit_updated_check_items_by_user([(props.struct_obj).key], check.value, (props.struct_obj).indeterminate)
}
// このアイテムがクリックされたときに呼び出されます。
// このアイテムのみにチェックが入るように上にemitします。
function click_item_by_user() {
    emit_click_item_by_user((props.struct_obj).key)
}
// このアイテムがクリックされたときに呼び出されます。
// このアイテム内のアイテムのみにチェックが入るように上にemitします。
function click_group_by_user() {
    let items = new Array<string>()
    let f = (struct: FoldableStructModel) => { }
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
    emit_click_items_by_user(items)
}
// 現在チェックの入っているアイテム名を配列で取得します。
function get_selected_items(): Array<string> {
    let items = new Array<string>()
    let f = (struct: FoldableStructModel) => { }
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
</script>

<style>
.tree_item {
    min-width: 200px;
    cursor: default;
}
</style>