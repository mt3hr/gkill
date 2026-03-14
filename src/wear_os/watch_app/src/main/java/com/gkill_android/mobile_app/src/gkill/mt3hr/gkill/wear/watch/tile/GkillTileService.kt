package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import androidx.concurrent.futures.CallbackToFutureAdapter
import androidx.wear.protolayout.ActionBuilders
import androidx.wear.protolayout.DeviceParametersBuilders
import androidx.wear.protolayout.LayoutElementBuilders
import androidx.wear.protolayout.ModifiersBuilders
import androidx.wear.protolayout.TimelineBuilders
import androidx.wear.protolayout.material.Chip
import androidx.wear.protolayout.material.CompactChip
import androidx.wear.protolayout.material.layouts.PrimaryLayout
import androidx.wear.protolayout.ResourceBuilders
import androidx.wear.tiles.RequestBuilders
import androidx.wear.tiles.TileBuilders
import androidx.wear.tiles.TileService
import com.google.common.util.concurrent.ListenableFuture
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.EXTRA_MODE
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.MODE_PLAING
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.MODE_RECORD
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.MainActivity

class GkillTileService : TileService() {

    override fun onTileRequest(requestParams: RequestBuilders.TileRequest): ListenableFuture<TileBuilders.Tile> =
        CallbackToFutureAdapter.getFuture { completer ->
            completer.set(buildTile(requestParams))
            "tileRequest"
        }

    override fun onTileResourcesRequest(requestParams: RequestBuilders.ResourcesRequest): ListenableFuture<ResourceBuilders.Resources> =
        CallbackToFutureAdapter.getFuture { completer ->
            completer.set(ResourceBuilders.Resources.Builder().setVersion("1").build())
            "resourcesRequest"
        }

    private fun buildTile(requestParams: RequestBuilders.TileRequest): TileBuilders.Tile {
        val deviceParams = requestParams.deviceConfiguration
        return TileBuilders.Tile.Builder()
            .setResourcesVersion("1")
            .setTileTimeline(
                TimelineBuilders.Timeline.fromLayoutElement(buildLayout(deviceParams))
            )
            .build()
    }

    private fun buildLayout(deviceParams: DeviceParametersBuilders.DeviceParameters): LayoutElementBuilders.LayoutElement {
        val launchRecord = ModifiersBuilders.Clickable.Builder()
            .setOnClick(
                ActionBuilders.LaunchAction.Builder()
                    .setAndroidActivity(
                        ActionBuilders.AndroidActivity.Builder()
                            .setPackageName(packageName)
                            .setClassName(MainActivity::class.java.name)
                            .addKeyToExtraMapping(
                                EXTRA_MODE,
                                ActionBuilders.AndroidStringExtra.Builder()
                                    .setValue(MODE_RECORD)
                                    .build()
                            )
                            .build()
                    ).build()
            ).build()

        val launchPlaing = ModifiersBuilders.Clickable.Builder()
            .setOnClick(
                ActionBuilders.LaunchAction.Builder()
                    .setAndroidActivity(
                        ActionBuilders.AndroidActivity.Builder()
                            .setPackageName(packageName)
                            .setClassName(MainActivity::class.java.name)
                            .addKeyToExtraMapping(
                                EXTRA_MODE,
                                ActionBuilders.AndroidStringExtra.Builder()
                                    .setValue(MODE_PLAING)
                                    .build()
                            )
                            .build()
                    ).build()
            ).build()

        val launchRefresh = ModifiersBuilders.Clickable.Builder()
            .setOnClick(
                ActionBuilders.LaunchAction.Builder()
                    .setAndroidActivity(
                        ActionBuilders.AndroidActivity.Builder()
                            .setPackageName(packageName)
                            .setClassName(TileRefreshActivity::class.java.name)
                            .build()
                    ).build()
            ).build()

        return PrimaryLayout.Builder(deviceParams)
            .setContent(
                LayoutElementBuilders.Column.Builder()
                    .addContent(
                        Chip.Builder(this, launchRecord, deviceParams)
                            .setPrimaryLabelContent("📝 記録する")
                            .build()
                    )
                    .addContent(
                        Chip.Builder(this, launchPlaing, deviceParams)
                            .setPrimaryLabelContent("▶ 実行中")
                            .build()
                    )
                    .build()
            )
            .setPrimaryChipContent(
                CompactChip.Builder(this, "🔄 更新", launchRefresh, deviceParams)
                    .build()
            )
            .build()
    }
}
