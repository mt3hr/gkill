<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="emits('requested_edit_rep_type', id)">
                <v-list-item-title>編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_rep_type', id)">
                <v-list-item-title>削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { computed, ref, type Ref } from 'vue';
import type { RepTypeStructContextMenuEmits } from './rep-type-struct-context-menu-emits';
import type { RepTypeStructContextMenuProps } from './rep-type-struct-context-menu-props';

const props = defineProps<RepTypeStructContextMenuProps>()
const emits = defineEmits<RepTypeStructContextMenuEmits>()
defineExpose({ show, hide })

const id: Ref<string> = ref("")
const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

async function show(e: MouseEvent, rep_type_id: string): Promise<void> {
    id.value = rep_type_id
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}
</script>
<style lang="css" scoped></style>