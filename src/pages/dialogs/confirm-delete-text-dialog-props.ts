'use strict';

import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface ConfirmDeleteTextDialogProps extends KyouViewPropsBase {
    text: Text;
    is_show: boolean;
}
