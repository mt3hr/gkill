<template>
    <div @dblclick="show_kyou_dialog()" @click.prevent="nextTick(() => emits('clicked_kyou', cloned_kyou))"
        :key="kyou.id" :class="'kyou_'.concat(kyou.id)">
        <div v-if="!show_content_only">
            <AttachedTag v-for="attached_tag in cloned_kyou.attached_tags" :tag="attached_tag" :key="attached_tag.id"
                :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
            <div v-if="show_attached_timeis">
                <AttachedTimeIsPlaing v-for="attached_timeis_plaing in cloned_kyou.attached_timeis_kyou"
                    :key="attached_timeis_plaing.id" :timeis_kyou="attached_timeis_plaing"
                    :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                    :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                    :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                    @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                    @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                    @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                    @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                    @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                    @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                    @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                    @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                    @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                    @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                    @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                    @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                    @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                    @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                    @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
                    @requested_reload_list="emits('requested_reload_list')"
                    @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
            </div>
            <v-row class="pa-0 ma-0" @contextmenu.prevent="async (e: any) => show_context_menu(e as PointerEvent)"
                :class="kyou_class">
                <v-col v-if="show_related_time" class="kyou_related_time pa-0 ma-0" cols="auto">
                    {{ related_time }}
                </v-col>
                <v-col v-if="show_update_time" class="kyou_update_time pa-0 ma-0" cols="auto">
                    {{ update_time }}
                </v-col>
                <v-spacer />
                <v-col v-if="show_rep_name" class="kyou_rep_name pa-0 ma-0" cols="auto">
                    <span>{{ rep_name }}</span>
                </v-col>
            </v-row>
        </div>
        <div :class="kyou_class">
            <KmemoView v-if="cloned_kyou.typed_kmemo" :kmemo="cloned_kyou.typed_kmemo" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width" :max-width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="kmemo_view" />
            <KCView v-if="cloned_kyou.typed_kc" :kc="cloned_kyou.typed_kc" :application_config="application_config"
                :draggable=draggable :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="kc_view" />
            <miKyouView v-if="cloned_kyou.typed_mi" :mi="cloned_kyou.typed_mi" :application_config="application_config"
                :draggable=draggable :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                :height="height" :width="width" :is_readonly_mi_check="is_readonly_mi_check"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="mi_view" />
            <NlogView v-if="cloned_kyou.typed_nlog" :nlog="cloned_kyou.typed_nlog" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="nlog_view" />
            <LantanaView v-if="cloned_kyou.typed_lantana" :lantana="cloned_kyou.typed_lantana" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="lantana_view" />
            <TimeIsView v-if="cloned_kyou.typed_timeis" :timeis="cloned_kyou.typed_timeis" :draggable=draggable
                :show_timeis_elapsed_time="show_timeis_elapsed_time"
                :show_timeis_plaing_end_button="show_timeis_plaing_end_button" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="highlight_targets" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :height="height" :width="width"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="timeis_view" />
            <URLogView v-if="cloned_kyou.typed_urlog" :urlog="cloned_kyou.typed_urlog" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="urlog_view" />
            <IDFKyouView v-if="cloned_kyou.typed_idf_kyou" :idf_kyou="cloned_kyou.typed_idf_kyou" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="idf_kyou_view" />
            <ReKyouView v-if="cloned_kyou.typed_rekyou" :rekyou="cloned_kyou.typed_rekyou" :draggable=draggable
                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :height="height" :width="width"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="rekyou_view" />
            <GitCommitLogView v-if="cloned_kyou.typed_git_commit_log" :git_commit_log="cloned_kyou.typed_git_commit_log"
                :draggable=draggable :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
                :height="height" :width="width" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                ref="git_commit_log_view" />
        </div>
        <div v-if="!show_content_only">
            <AttachedText v-for="attached_text in cloned_kyou.attached_texts" :text="attached_text"
                :key="attached_text.id" :application_config="application_config" :gkill_api="gkill_api"
                :kyou="cloned_kyou" :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </div>
        <div v-if="!show_content_only">
            <AttachedNotification v-for="attached_notification in cloned_kyou.attached_notifications"
                :key="attached_notification.id" :notification="attached_notification"
                :application_config="application_config" :gkill_api="gkill_api" :kyou="cloned_kyou"
                :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </div>
        <kyouDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            :is_readonly_mi_check="is_readonly_mi_check" :show_timeis_plaing_end_button="show_timeis_plaing_end_button"
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
            ref="kyou_dialog" />
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { computed, watch, type Ref, ref, nextTick, onUnmounted } from 'vue'
import { format_time } from '@/classes/format-date-time'

import AttachedTag from './attached-tag.vue'
import AttachedText from './attached-text.vue'
import AttachedTimeIsPlaing from './attached-time-is-plaing.vue'
import AttachedNotification from './attached-notification.vue'
import GitCommitLogView from './git-commit-log-view.vue'
import IDFKyouView from './idf-kyou-view.vue'
import KmemoView from './kmemo-view.vue'
import KCView from './kc-view.vue'
import LantanaView from './lantana-view.vue'
import miKyouView from './mi-kyou-view.vue'
import NlogView from './nlog-view.vue'
import ReKyouView from './re-kyou-view.vue'
import TimeIsView from './time-is-view.vue'
import URLogView from './ur-log-view.vue'
import kyouDialog from '../dialogs/kyou-dialog.vue'

import type { KyouViewEmits } from './kyou-view-emits'
import type { KyouViewProps } from './kyou-view-props'
import { Kyou } from '@/classes/datas/kyou'
import type MiKyouView from './mi-kyou-view.vue'
import type UrLogView from './ur-log-view.vue'
import type IdfKyouView from './idf-kyou-view.vue'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import moment from 'moment'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const kyou_dialog = ref<InstanceType<typeof kyouDialog> | null>(null);
const kmemo_view = ref<InstanceType<typeof KmemoView> | null>(null);
const kc_view = ref<InstanceType<typeof KCView> | null>(null);
const mi_view = ref<InstanceType<typeof MiKyouView> | null>(null);
const nlog_view = ref<InstanceType<typeof NlogView> | null>(null);
const lantana_view = ref<InstanceType<typeof LantanaView> | null>(null);
const timeis_view = ref<InstanceType<typeof TimeIsView> | null>(null);
const urlog_view = ref<InstanceType<typeof UrLogView> | null>(null);
const idf_kyou_view = ref<InstanceType<typeof IdfKyouView> | null>(null);
const rekyou_view = ref<InstanceType<typeof ReKyouView> | null>(null);
const git_commit_log_view = ref<InstanceType<typeof GitCommitLogView> | null>(null);

const props = defineProps<KyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())

onUnmounted(() => {
    cloned_kyou.value.abort_controller.abort()
    cloned_kyou.value.abort_controller = new AbortController()
})

const related_time = computed(() => format_time(props.kyou.related_time))
const update_time = computed(() => format_time(props.kyou.update_time))
const rep_name = computed(() => props.kyou.rep_name)

watch(() => props.kyou, async () => {
    cloned_kyou.value.abort_controller.abort()
    cloned_kyou.value = props.kyou.clone()
    cloned_kyou.value.abort_controller = new AbortController()
    if (props.force_show_latest_kyou_info) {
        await cloned_kyou.value.reload(true, props.force_show_latest_kyou_info);//最新を読み込むためにReload
    }
    (() => load_attached_infos())(); //非同期で実行してほしい
});

(async () => {
    if (props.force_show_latest_kyou_info) {
        await cloned_kyou.value.reload(true, props.force_show_latest_kyou_info);//最新を読み込むためにReload
    }
    load_attached_infos()
})(); //非同期で実行してほしい


const kyou_class = computed(() => {
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
        return "highlighted_kyou"
    }
    return ""
})

async function load_attached_infos(): Promise<void> {
    try {
        const awaitPromises = new Array<Promise<any>>()
        try {
            awaitPromises.push(cloned_kyou.value.load_typed_datas())
            if (props.show_attached_tags) {
                awaitPromises.push(cloned_kyou.value.load_attached_tags())
            }
            if (props.show_attached_texts) {
                awaitPromises.push(cloned_kyou.value.load_attached_texts())
            }
            if (props.show_attached_notifications) {
                awaitPromises.push(cloned_kyou.value.load_attached_notifications())
            }
            if (props.show_attached_timeis) {
                awaitPromises.push(cloned_kyou.value.load_attached_timeis())
            }
            await Promise.all(awaitPromises)
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    } catch (err: any) {
        // abortは握りつぶす
        if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            // abort以外はエラー出力する
            console.error(err)
        }
    }
}


async function show_context_menu(e: PointerEvent): Promise<void> {
    if (!props.enable_context_menu) {
        return
    }
    kmemo_view.value?.show_context_menu(e)
    kc_view.value?.show_context_menu(e)
    mi_view.value?.show_context_menu(e)
    nlog_view.value?.show_context_menu(e)
    lantana_view.value?.show_context_menu(e)
    timeis_view.value?.show_context_menu(e)
    urlog_view.value?.show_context_menu(e)
    idf_kyou_view.value?.show_context_menu(e)
    rekyou_view.value?.show_context_menu(e)
    git_commit_log_view.value?.show_context_menu(e)
}

function show_kyou_dialog(): void {
    if (props.enable_dialog) {
        kyou_dialog.value?.show()
    }
}

</script>
<style lang="css" scoped>
.highlighted_kyou>* {
    background-color: rgb(var(--v-theme-highlight));
}

.kyou_related_time,
.kyou_update_time,
.kyou_rep_name,
.kyou_device,
.kyou_data_type {
    font-size: small;
    color: silver;
}
</style>
