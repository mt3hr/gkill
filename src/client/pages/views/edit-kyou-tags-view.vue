<template>
    <div class="gkill-kyou-tags">
        <!-- 既存タグ。削除は保存を押すまで確定しないので、押し間違えても戻せる。
             一覧の attached-tag.vue(表示用チップ、右クリックで独自メニュー)とは別物にしてある。
             あれを埋めると「削除」の導線がここと二重になる -->
        <div v-if="existing_tags.length !== 0" class="gkill-kyou-tags__chips">
            <v-chip v-for="tag in existing_tags" :key="tag.id" size="small" label class="ma-1"
                :class="is_removed(tag) ? 'gkill-kyou-tags__chip--removed' : ''"
                :disabled="is_readonly" @click="toggle_remove(tag)">
                {{ tag.tag }}
                <v-icon end size="small">{{ is_removed(tag) ? 'mdi-restore' : 'mdi-close' }}</v-icon>
            </v-chip>
        </div>
        <v-text-field class="input text" type="text" v-model="tag_names_text"
            :label="i18n.global.t('TAG_TITLE')" :readonly="is_readonly" hide-details />
        <div v-if="tag_history.length !== 0" class="gkill-kyou-tags__history">
            <span class="gkill-kyou-tags__history-label">{{ i18n.global.t("ADD_TAG_FROM_HISTORY_TITLE") }}</span>
            <v-chip v-for="history_tag in tag_history" :key="history_tag" size="small" label class="ma-1"
                :disabled="is_readonly" @click="append_history_tag(history_tag)">
                {{ history_tag }}
            </v-chip>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditKyouTagsViewProps } from './edit-kyou-tags-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useEditKyouTagsView } from '@/classes/use-edit-kyou-tags-view'

const props = defineProps<EditKyouTagsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    tag_names_text,
    existing_tags,
    tag_history,

    // Business logic / template handlers
    is_removed,
    toggle_remove,
    append_history_tag,

    // Exposed to parent
    get_tag_names,
    get_removed_tags,
    has_pending_changes,
    reset,
} = useEditKyouTagsView({ props, emits })

defineExpose({ get_tag_names, get_removed_tags, has_pending_changes, reset })
</script>
<style lang="css" scoped>
/* 履歴チップが多いときに横へ流れて親を広げないよう折り返す */
.gkill-kyou-tags__chips,
.gkill-kyou-tags__history {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
}

.gkill-kyou-tags__history-label {
    font-size: small;
    opacity: 0.7;
}

/* 削除マークが付いた既存タグ。保存するまでは消えていないので、
   消えたように見せず「消す予定」だと分かる見た目にする */
.gkill-kyou-tags__chip--removed {
    text-decoration: line-through;
    opacity: 0.5;
}
</style>
