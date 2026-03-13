<template>
    <tr v-if="is_item()" :draggable="is_editable" @dragstart="drag_start" @drop="drop" :dropzone="is_editable"
        :key="props.struct_obj.key" @dragover="dragover" class="foldable_struct_draggable"
        @contextmenu.prevent.stop="onContextmenuItem">
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
        :key="props.struct_obj.key" @dragover="dragover" class="foldable_struct_draggable"
        @contextmenu.prevent.stop="onContextmenuItem">
        <td>
            <table>
                <tr>
                    <td v-if="is_show_checkbox">
                        <input type="checkbox" class="checkbox_in_foldable_struct" v-model="check"
                            @change="change_group_by_user" :indeterminate="indeterminate_group" />
                    </td>
                    <td>
                        <span v-if="open_group" style="cursor: default" @click="onToggleOpenGroup">▽</span>
                        <span v-if="!open_group" style="cursor: default" @click="onToggleOpenGroup">▷</span>
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
                    :is_root="false"
                    @received_errors="onChildReceivedErrors"
                    @received_messages="onChildReceivedMessages"
                    @dblclicked_item="onChildDblclickedItem"
                    @contextmenu_item.prevent.stop="onChildContextmenuItem"
                    @requested_update_check_state="onChildRequestedUpdateCheckState"
                    @requested_move_struct_obj="handle_move_struct_obj" @requested_update_struct_obj="update_struct_obj"
                    ref="child_foldable_structs" />
            </table>
        </td>
    </tr>
</template>
<script lang="ts" setup>
import FoldableStruct from './foldable-struct.vue'
import type { FoldableStructEmits } from './foldable-struct-emits'
import type { FoldableStructProps } from './foldable-struct-props'
import { useFoldableStruct } from '@/classes/use-foldable-struct'

const props = defineProps<FoldableStructProps>()
const emits = defineEmits<FoldableStructEmits>()

const {
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
} = useFoldableStruct({ props, emits })

defineExpose({ get_selected_items, handle_move_struct_obj, get_foldable_struct, delete_struct, update_check })
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

.foldable_struct_draggable {
    user-select: none;
    touch-action: none;
}
</style>
