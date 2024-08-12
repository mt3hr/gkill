<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { DeviceStructElement } from './device-struct-element.vue';
import { EditDeviceStructViewEmits } from './edit-device-struct-view-emits';
import { EditDeviceStructViewProps } from './edit-device-struct-view-props';
import { TagStructElement } from './tag-struct-element';

// ˄

export class EditDeviceStructView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    add_new_device_struct_element_dialog: AddNewDeviceStructElementDialog;

    edit_device_struct_element_dialog: EditDeviceStructElementDialog;

    private props: EditDeviceStructViewProps;

    private emits: EditDeviceStructViewEmits;

    private tagStructElement: TagStructElement;

    private device_struct_root_element: DeviceStructElement;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>