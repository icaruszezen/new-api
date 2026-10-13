package com.newapi.admin.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * shadcn/ui "neutral" design tokens. Components read them through [AppTheme.colors]; Material components
 * get them through the colour scheme that [AdminTheme] derives from them.
 */
@Immutable
data class AppColors(
    val background: Color,
    val foreground: Color,
    val card: Color,
    val popover: Color,
    val primary: Color,
    val primaryForeground: Color,
    val secondary: Color,
    val secondaryForeground: Color,
    val muted: Color,
    val mutedForeground: Color,
    val accent: Color,
    val accentForeground: Color,
    val destructive: Color,
    val border: Color,
    val input: Color,
    val ring: Color,
    val isDark: Boolean,
)

// The window background in res/values*/colors.xml matches `background` of each palette.

val LightAppColors = AppColors(
    background = Color(0xFFFFFFFF),
    foreground = Color(0xFF0A0A0A),
    card = Color(0xFFFFFFFF),
    popover = Color(0xFFFFFFFF),
    primary = Color(0xFF171717),
    primaryForeground = Color(0xFFFAFAFA),
    secondary = Color(0xFFF5F5F5),
    secondaryForeground = Color(0xFF171717),
    muted = Color(0xFFF5F5F5),
    mutedForeground = Color(0xFF737373),
    accent = Color(0xFFF5F5F5),
    accentForeground = Color(0xFF171717),
    destructive = Color(0xFFDC2626),
    border = Color(0xFFE5E5E5),
    input = Color(0xFFE5E5E5),
    ring = Color(0xFFA1A1A1),
    isDark = false,
)

val DarkAppColors = AppColors(
    background = Color(0xFF0A0A0A),
    foreground = Color(0xFFFAFAFA),
    card = Color(0xFF171717),
    popover = Color(0xFF171717),
    primary = Color(0xFFE5E5E5),
    primaryForeground = Color(0xFF171717),
    secondary = Color(0xFF262626),
    secondaryForeground = Color(0xFFFAFAFA),
    muted = Color(0xFF262626),
    mutedForeground = Color(0xFFA1A1A1),
    accent = Color(0xFF262626),
    accentForeground = Color(0xFFFAFAFA),
    destructive = Color(0xFFFF6467),
    // Translucent white, so borders read the same on the page and on cards.
    border = Color(0x1AFFFFFF),
    input = Color(0x26FFFFFF),
    ring = Color(0xFF737373),
    isDark = true,
)

val LocalAppColors = staticCompositionLocalOf { LightAppColors }

object AppTheme {
    val colors: AppColors
        @Composable @ReadOnlyComposable
        get() = LocalAppColors.current
}

/** Semantic colour of a status pill or metric. */
enum class Tone { Success, Warning, Danger, Info, Neutral }

data class ToneColors(val container: Color, val content: Color)

@Composable
@ReadOnlyComposable
fun Tone.colors(): ToneColors {
    val colors = LocalAppColors.current
    val dark = colors.isDark
    return when (this) {
        // Tailwind emerald / amber / red / blue: a translucent tint behind 700 (light) or 400 (dark) text.
        Tone.Success -> ToneColors(Color(0xFF10B981).copy(alpha = if (dark) 0.16f else 0.12f), if (dark) Color(0xFF34D399) else Color(0xFF047857))
        Tone.Warning -> ToneColors(Color(0xFFF59E0B).copy(alpha = if (dark) 0.16f else 0.14f), if (dark) Color(0xFFFBBF24) else Color(0xFFB45309))
        Tone.Danger -> ToneColors(Color(0xFFEF4444).copy(alpha = if (dark) 0.16f else 0.10f), if (dark) Color(0xFFF87171) else Color(0xFFDC2626))
        Tone.Info -> ToneColors(Color(0xFF3B82F6).copy(alpha = if (dark) 0.16f else 0.10f), if (dark) Color(0xFF60A5FA) else Color(0xFF1D4ED8))
        Tone.Neutral -> ToneColors(colors.secondary, colors.secondaryForeground)
    }
}
