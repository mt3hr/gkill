<template>
    <span v-if="kyou.typed_timeis" :class="plaing_class"
        @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
        {{ kyou.typed_timeis.title }}
    </span>
    <AttachedTimeIsPlaingContextMenu :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
        v-if="kyou.typed_timeis" :timeis="kyou.typed_timeis" :last_added_tag="last_added_tag"
        :highlight_targets="highlight_targets" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref } from 'vue'
import AttachedTimeIsPlaingContextMenu from './attached-timeis-plaing-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTimeIsPlaingContextMenu> | null>(null);

const props = defineProps<AttachedTimeIsPlaingProps>()
const emits = defineEmits<KyouViewEmits>()

const plaing_class = computed(() => {
    if (!props.kyou) {
        return ""
    }
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.kyou.id
            && props.highlight_targets[i].create_time.getTime() === props.kyou.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.kyou.update_time.getTime()) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_plaing"
    }
    return "plaing"
})


async function show_context_menu(e: PointerEvent): Promise<void> {
    context_menu.value?.show(e)
}

</script>
<style scoped>
.plaing {
    /* タグとの合わせ */
    position: relative;
    top: 6px;


    display: inline-flex;
    border: solid white 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    font-size: small;
    background: lightgray
}

.plaing::after {
    content: "";
    background: white;
    border-top: 9.5px solid transparent;
    border-left: 10px solid #d3d3d3;
    border-bottom: 9.5px solid transparent;
}

.plaing::before {
    content: "";
    background: white;
    border-top: 9.5px solid transparent;
    border-right: 10px solid #d3d3d3;
    border-bottom: 9.5px solid transparent;
}

.highlighted_plaing {
    /* タグとの合わせ */
    position: relative;
    top: 6px;

    display: inline-flex;
    border: solid white 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    font-size: small;
    background: lightgreen;
}

.highlighted_plaing::after {
    content: "";
    background: white;
    border-top: 9.5px solid transparent;
    border-left: 10px solid lightgreen;
    border-bottom: 9.5px solid transparent;
}

.highlighted_plaing::before {
    content: "";
    background: white;
    border-top: 9.5px solid transparent;
    border-right: 10px solid lightgreen;
    border-bottom: 9.5px solid transparent;
}
</style>