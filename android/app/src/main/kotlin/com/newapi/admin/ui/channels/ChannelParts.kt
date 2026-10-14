package com.newapi.admin.ui.channels

import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import com.newapi.admin.R
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelTypes
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberCurrentUser
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
import java.util.Locale
import kotlinx.coroutines.flow.Flow

/** What the signed-in admin may do with channels (`admin_permissions.channel.*`; root can do everything). */
data class ChannelPermissions(
    /** Test, balance refresh, enable/disable, fetch upstream models. */
    val operate: Boolean = false,
    /** Edit non-sensitive fields. */
    val write: Boolean = false,
    /** Create, delete, and edit type/key/base URL. */
    val sensitiveWrite: Boolean = false,
)

@Composable
internal fun rememberChannelPermissions(): ChannelPermissions {
    val user = rememberCurrentUser()
    return remember(user) {
        ChannelPermissions(
            operate = user?.can("channel", "operate") == true,
            write = user?.can("channel", "write") == true,
            sensitiveWrite = user?.can("channel", "sensitive_write") == true,
        )
    }
}

/** Group names from `GET /api/group/`, loaded on demand. */
data class GroupOptions(
    val names: List<String> = emptyList(),
    val isLoading: Boolean = false,
    val failed: Boolean = false,
)

internal val Channel.displayName: String get() = name.ifBlank { "#$id" }

/** The single-channel test endpoint rejects these types. */
internal val Channel.isTestable: Boolean get() = type !in ChannelTypes.testUnsupported

/** Upstream balance can be queried for these types; multi-key channels are refused by the server. */
internal val Channel.hasBalance: Boolean get() = type in ChannelTypes.balanceSupported && !channelInfo.isMultiKey

/** Seconds with two decimals, like the web's channel timing labels. */
internal fun formatSeconds(seconds: Double): String = String.format(Locale.ROOT, "%.2f", seconds)

/** Web thresholds (`RESPONSE_TIME_CONFIG`): up to 1 s good, up to 2 s fair, slower poor. */
internal fun responseTimeTone(ms: Int): Tone = when {
    ms <= 0 -> Tone.Neutral
    ms <= 1000 -> Tone.Success
    ms <= 2000 -> Tone.Warning
    else -> Tone.Danger
}

/** Web `getBalanceVariant`: 0 neutral, under 1 USD danger, under 10 USD warning. */
internal fun balanceTone(usd: Double): Tone = when {
    usd.isNaN() || usd == 0.0 -> Tone.Neutral
    usd < 1 -> Tone.Danger
    usd < 10 -> Tone.Warning
    else -> Tone.Success
}

/** Type label as the web shows it: the labels its locales translate are string resources, the other (brand) names stay as in [ChannelTypes]. */
@Composable
internal fun channelTypeLabel(type: Int): String = when (type) {
    0 -> stringResource(R.string.common_unknown)
    8 -> stringResource(R.string.channels_type_name_custom)
    15 -> stringResource(R.string.channels_type_name_baidu)
    16 -> stringResource(R.string.channels_type_name_zhipu)
    17 -> stringResource(R.string.channels_type_name_ali)
    18 -> stringResource(R.string.channels_type_name_xunfei)
    23 -> stringResource(R.string.channels_type_name_tencent)
    26 -> stringResource(R.string.channels_type_name_zhipu_v4)
    45 -> stringResource(R.string.channels_type_name_volcengine)
    46 -> stringResource(R.string.channels_type_name_baidu_v2)
    58 -> stringResource(R.string.channels_type_name_advanced_custom)
    else -> ChannelTypes.nameOf(type)
}

@Composable
internal fun responseTimeText(ms: Int): String = when {
    ms <= 0 -> stringResource(R.string.channels_not_tested)
    ms < 1000 -> stringResource(R.string.channels_response_ms, ms)
    else -> stringResource(R.string.channels_response_s, formatSeconds(ms / 1000.0))
}

/** Text colour for a metric; neutral values keep the surrounding colour. */
@Composable
internal fun toneTextColor(tone: Tone): Color =
    if (tone == Tone.Neutral) AppTheme.colors.mutedForeground else tone.colors().content

@Composable
internal fun ChannelStatusPill(status: Int, modifier: Modifier = Modifier) {
    val (label, tone) = when (status) {
        ChannelStatus.ENABLED -> R.string.common_enabled to Tone.Success
        ChannelStatus.MANUALLY_DISABLED -> R.string.common_disabled to Tone.Danger
        ChannelStatus.AUTO_DISABLED -> R.string.channels_status_auto_disabled to Tone.Warning
        else -> R.string.common_unknown to Tone.Neutral
    }
    StatusPill(stringResource(label), tone, modifier)
}

/** Shows each one-off ViewModel message once in [hostState]; a newer message replaces the one on screen. */
@Composable
internal fun ChannelMessagesEffect(messages: Flow<UiMessage>, hostState: SnackbarHostState) {
    var current by remember { mutableStateOf<UiMessage?>(null) }
    LaunchedEffect(messages) {
        messages.collect { current = it }
    }
    current?.let { message ->
        val text = message.text()
        LaunchedEffect(message) {
            hostState.showSnackbar(text)
            current = null
        }
    }
}
