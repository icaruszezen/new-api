package com.newapi.admin.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

/** Semantic colour of a status pill or metric. */
enum class Tone { Success, Warning, Danger, Info, Neutral }

data class ToneColors(val container: Color, val content: Color)

@Composable
fun Tone.colors(): ToneColors {
    val dark = isSystemInDarkTheme()
    return when (this) {
        Tone.Success -> if (dark) ToneColors(Color(0xFF1E3A24), Color(0xFF8BD69B)) else ToneColors(Color(0xFFDDF3E2), Color(0xFF1B6B33))
        Tone.Warning -> if (dark) ToneColors(Color(0xFF3D3015), Color(0xFFF2C46B)) else ToneColors(Color(0xFFFFF0CC), Color(0xFF8A5A00))
        Tone.Danger -> if (dark) ToneColors(Color(0xFF45201F), Color(0xFFFFA8A3)) else ToneColors(Color(0xFFFDE2E0), Color(0xFFB3261E))
        Tone.Info -> if (dark) ToneColors(Color(0xFF1C2E45), Color(0xFF9CC4FF)) else ToneColors(Color(0xFFDDE9FF), Color(0xFF1D4F91))
        Tone.Neutral -> if (dark) ToneColors(Color(0xFF2E3034), Color(0xFFC4C7CC)) else ToneColors(Color(0xFFECEEF1), Color(0xFF4A4F57))
    }
}

/** Brand seed colours used when dynamic colour is unavailable (Android < 12). */
object BrandColors {
    val Primary = Color(0xFF3557D6)
    val PrimaryDark = Color(0xFFB7C4FF)
}
