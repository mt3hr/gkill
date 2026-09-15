<template>
    <div class="alert_container" role="status" aria-live="polite">
        <v-slide-y-transition group>
            <v-alert v-for="item in feed_items" :key="item.id" :color="alert_color(item.level)"
                :role="alert_role(item.level)" :closable="item.closable" density="compact"
                @click:close="onClickClose(item.id)">
                <div class="gkill_feed_message">
                    {{ item.message }}
                    <span v-if="item.count > 1" class="gkill_feed_count">×{{ item.count }}</span>
                </div>
                <div v-if="item.hint" class="gkill_feed_hint">{{ item.hint }}</div>
                <div v-if="show_footer(item)" class="gkill_feed_footer">
                    <span class="gkill_feed_code">{{ item.code }}<template v-if="item.reason"> · {{ item.reason }}</template></span>
                    <v-btn size="x-small" variant="text" icon="mdi-content-copy" class="gkill_feed_copy"
                        :title="$t('ERROR_ALERT_COPY_TITLE')" :aria-label="$t('ERROR_ALERT_COPY_TITLE')"
                        @click="onClickCopy(item)" />
                </div>
            </v-alert>
        </v-slide-y-transition>
    </div>
</template>
<script setup lang="ts">
import { useGkillMessageFeedView } from '@/classes/use-gkill-message-feed-view'

const {
    // State
    feed_items,

    // Template helpers
    alert_color,
    alert_role,
    show_footer,

    // Event handlers
    onClickClose,
    onClickCopy,
} = useGkillMessageFeedView()
</script>
<style lang="css" scoped>
.alert_container>div {
    width: fit-content;
}

.alert_container {
    justify-items: end;
    position: fixed;
    top: 60px;
    right: 10px;
    display: grid;
    grid-gap: .5em;
    z-index: 99;
    max-width: min(480px, calc(100vw - 20px));
}

.gkill_feed_message {
    white-space: pre-line;
    overflow-wrap: anywhere;
}

.gkill_feed_count {
    margin-left: .5em;
    font-size: small;
    opacity: .8;
}

.gkill_feed_hint {
    margin-top: .25em;
    font-size: small;
    opacity: .9;
    overflow-wrap: anywhere;
}

.gkill_feed_footer {
    margin-top: .25em;
    display: flex;
    align-items: center;
    gap: .25em;
    font-size: x-small;
    opacity: .8;
    font-family: monospace;
}
</style>
