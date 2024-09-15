'use strict';

import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface EditTextDialogProps extends KyouViewPropsBase {
    text: Text;
    is_show: boolean;
}
