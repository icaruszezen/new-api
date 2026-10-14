package com.newapi.admin.ui.update

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.OpenInNew
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.update.UpdateDownload
import com.newapi.admin.update.installApk
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppDialog
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import java.io.File

/**
 * The "update available" dialog, shown over any screen while [com.newapi.admin.update.UpdateState.promptVisible]:
 * release notes, then download progress, then the system installer.
 */
@Composable
fun UpdatePrompt() {
    val container = LocalAppContainer.current
    val updates = container.updates
    // Collected only while started, so a download that finishes in the background opens the
    // installer when the app comes back instead of from the background, where Android blocks it.
    val state by updates.state.collectAsStateWithLifecycle()
    val release = state.available
    if (!state.promptVisible || release == null) return

    val context = LocalContext.current
    val uriHandler = LocalUriHandler.current
    val colors = AppTheme.colors
    val download = state.download
    val ready = download as? UpdateDownload.Ready
    var installerMissing by remember(ready) { mutableStateOf(false) }
    // Saved so that recreating the activity (rotation, dark mode) does not open the installer again.
    var installerOpened by rememberSaveable(ready?.apk?.path) { mutableStateOf(false) }
    val install = { apk: File ->
        if (apk.exists()) installerMissing = !context.installApk(apk) else updates.forgetDownload()
    }
    LaunchedEffect(ready) {
        if (ready != null && !installerOpened) {
            installerOpened = true
            install(ready.apk)
        }
    }
    val openInBrowser = {
        // No browser at all: nothing more the app can offer.
        runCatching { uriHandler.openUri(release.pageUrl) }
        Unit
    }

    AppDialog(
        onDismissRequest = updates::dismissPrompt,
        title = stringResource(R.string.update_dialog_title),
        confirmButton = {
            when (download) {
                null -> AppButton(stringResource(R.string.update_now), onClick = updates::startDownload)
                is UpdateDownload.Running -> AppButton(stringResource(R.string.update_downloading), onClick = {}, loading = true)
                is UpdateDownload.Failed -> AppButton(stringResource(R.string.common_retry), onClick = updates::startDownload)
                is UpdateDownload.Ready -> AppButton(stringResource(R.string.update_install), onClick = { install(download.apk) })
            }
        },
        dismissButton = {
            if (download is UpdateDownload.Running) {
                AppButton(stringResource(R.string.common_cancel), onClick = updates::cancelDownload, variant = ButtonVariant.Outline)
            } else {
                AppButton(
                    stringResource(if (download == null) R.string.update_later else R.string.common_close),
                    onClick = updates::dismissPrompt,
                    variant = ButtonVariant.Outline,
                )
            }
        },
    ) {
        // Scrolls on short windows (landscape, large fonts), so the alerts and their actions stay reachable.
        Column(Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Column {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(release.versionName, style = MaterialTheme.typography.titleMedium)
                    if (release.isBeta) {
                        Spacer(Modifier.width(8.dp))
                        StatusPill(stringResource(R.string.update_beta_badge), Tone.Warning)
                    }
                }
                Text(
                    stringResource(R.string.update_version_detail, container.appVersion, Formats.fileSize(release.apkSize)),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedForeground,
                )
            }
            if (release.notes.isNotEmpty()) {
                Text(
                    release.notes,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = 180.dp)
                        .background(colors.muted, MaterialTheme.shapes.medium)
                        .verticalScroll(rememberScrollState())
                        .padding(12.dp),
                )
            }
            when (download) {
                null -> Text(stringResource(R.string.update_install_hint), style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground)
                is UpdateDownload.Running -> Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    if (download.totalBytes > 0) {
                        LinearProgressIndicator(
                            progress = { (download.downloadedBytes.toFloat() / download.totalBytes).coerceIn(0f, 1f) },
                            modifier = Modifier.fillMaxWidth(),
                            color = colors.primary,
                            trackColor = colors.muted,
                            drawStopIndicator = {},
                        )
                    } else {
                        LinearProgressIndicator(Modifier.fillMaxWidth(), color = colors.primary, trackColor = colors.muted)
                    }
                    Text(
                        if (download.totalBytes > 0) {
                            stringResource(R.string.update_progress, Formats.fileSize(download.downloadedBytes), Formats.fileSize(download.totalBytes))
                        } else {
                            Formats.fileSize(download.downloadedBytes)
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.mutedForeground,
                    )
                }
                is UpdateDownload.Failed -> AppAlert(
                    text = errorMessage(download.error),
                    tone = Tone.Danger,
                    icon = Icons.Outlined.ErrorOutline,
                    action = {
                        AppButton(
                            stringResource(R.string.update_open_in_browser),
                            onClick = openInBrowser,
                            variant = ButtonVariant.Ghost,
                            size = ButtonSize.Small,
                            leadingIcon = Icons.AutoMirrored.Outlined.OpenInNew,
                        )
                    },
                )
                is UpdateDownload.Ready -> if (installerMissing) {
                    AppAlert(
                        text = stringResource(R.string.update_installer_missing),
                        tone = Tone.Warning,
                        action = {
                            AppButton(
                                stringResource(R.string.update_open_in_browser),
                                onClick = openInBrowser,
                                variant = ButtonVariant.Ghost,
                                size = ButtonSize.Small,
                                leadingIcon = Icons.AutoMirrored.Outlined.OpenInNew,
                            )
                        },
                    )
                } else {
                    Text(stringResource(R.string.update_ready), style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground)
                }
            }
        }
    }
}
