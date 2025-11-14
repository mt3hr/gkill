<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_dnote_item_list', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_dnote_item_list', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue';
import type { DnoteItemListContextMenuEmits } from './dnote-item-list-context-menu-emits';
import type { DnoteItemListContextMenuProps } from './dnote-item-list-context-menu-props';

const props = defineProps<DnoteItemListContextMenuProps>()
const emits = defineEmits<DnoteItemListContextMenuEmits>()
defineExpose({ show, hide })

const id: Ref<string> = ref("")
const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 2))), position_y.value.valueOf())}px; }`)

async function show(e: MouseEvent, dnote_item_id: string): Promise<void> {
    id.value = dnote_item_id
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}
</script>
<style lang="css" scoped></style>