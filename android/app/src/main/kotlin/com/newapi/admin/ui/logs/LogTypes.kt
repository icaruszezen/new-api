package com.newapi.admin.ui.logs

import com.newapi.admin.R
import com.newapi.admin.core.model.LogType
import com.newapi.admin.ui.theme.Tone

/** Log types the server writes, in the web's filter order (type 0 is the "all" filter, not a choice). */
internal val filterableLogTypes = listOf(
    LogType.TOPUP,
    LogType.CONSUME,
    LogType.MANAGE,
    LogType.SYSTEM,
    LogType.ERROR,
    LogType.REFUND,
    LogType.LOGIN,
)

/** Web labels (usage-logs/constants.ts LOG_TYPES); a row with type 0 or an unknown value shows "Unknown". */
internal fun logTypeLabel(type: Int): Int = when (type) {
    LogType.TOPUP -> R.string.logs_type_topup
    LogType.CONSUME -> R.string.logs_type_consume
    LogType.MANAGE -> R.string.logs_type_manage
    LogType.SYSTEM -> R.string.logs_type_system
    LogType.ERROR -> R.string.logs_type_error
    LogType.REFUND -> R.string.logs_type_refund
    LogType.LOGIN -> R.string.logs_type_login
    else -> R.string.logs_type_unknown
}

internal fun logTypeTone(type: Int): Tone = when (type) {
    LogType.CONSUME -> Tone.Success
    LogType.TOPUP, LogType.REFUND -> Tone.Info
    LogType.MANAGE -> Tone.Warning
    LogType.ERROR -> Tone.Danger
    else -> Tone.Neutral
}

/** Types whose rows carry model, token, token counts and cost (web: 0, 2, 5, 6). */
internal fun hasUsage(type: Int): Boolean =
    type == LogType.ALL || type == LogType.CONSUME || type == LogType.ERROR || type == LogType.REFUND

/** Types whose rows carry stream flag and timings (web: 2, 5). */
internal fun hasTiming(type: Int): Boolean = type == LogType.CONSUME || type == LogType.ERROR

/** Web timing colours: first token green < 5 s, amber < 10 s. */
internal fun firstResponseTone(seconds: Double): Tone = when {
    seconds < 5 -> Tone.Success
    seconds < 10 -> Tone.Warning
    else -> Tone.Danger
}

/** Web total-time colour: by duration for short answers, by throughput from 100 completion tokens on. */
internal fun useTimeTone(useTimeSeconds: Int, completionTokens: Long): Tone {
    if (completionTokens < 100 || useTimeSeconds <= 0) {
        return when {
            useTimeSeconds < 10 -> Tone.Success
            useTimeSeconds < 30 -> Tone.Warning
            else -> Tone.Danger
        }
    }
    val tokensPerSecond = completionTokens.toDouble() / useTimeSeconds
    return when {
        tokensPerSecond >= 30 -> Tone.Success
        tokensPerSecond >= 15 -> Tone.Warning
        else -> Tone.Danger
    }
}
