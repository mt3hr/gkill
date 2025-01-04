<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="emits('requested_edit_kftl_template', id)">
                <v-list-item-title>編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_kftl_template', id)">
                <v-list-item-title>削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { computed, ref, type Ref } from 'vue';
import type { KFTLTemplateStructContextMenuEmits } from './kftl_template-struct-context-menu-emits';
import type { KFTLTemplateStructContextMenuProps } from './kftl_template-struct-context-menu-props';

defineProps<KFTLTemplateStructContextMenuProps>()
const emits = defineEmits<KFTLTemplateStructContextMenuEmits>()
defineExpose({ show, hide })

const id: Ref<string> = ref("")
const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(document.defaultView!.innerHeight - 400, position_y.value.valueOf())}px; }`)

async function show(e: MouseEvent, kftl_template_id: string): Promise<void> {
    id.value = kftl_template_id
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}
</script>
<style lang="css" scoped></style>