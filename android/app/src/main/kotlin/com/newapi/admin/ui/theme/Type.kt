package com.newapi.admin.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import com.newapi.admin.R

// Geist (SIL OFL 1.1, see assets/licenses/geist-OFL.txt). Characters it lacks, such as CJK, fall back to the system font.
val Geist = FontFamily(
    Font(R.font.geist_regular, FontWeight.Normal),
    Font(R.font.geist_medium, FontWeight.Medium),
    Font(R.font.geist_semibold, FontWeight.SemiBold),
)

/** Monospace face for timestamps, keys and raw JSON. */
val GeistMono = FontFamily(Font(R.font.geist_mono_regular, FontWeight.Normal))

private val Base = TextStyle(fontFamily = Geist, letterSpacing = 0.sp)

val AppTypography = Typography(
    displayLarge = Base.copy(fontSize = 48.sp, lineHeight = 56.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-1).sp),
    displayMedium = Base.copy(fontSize = 36.sp, lineHeight = 44.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.8).sp),
    displaySmall = Base.copy(fontSize = 30.sp, lineHeight = 38.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.6).sp),
    headlineLarge = Base.copy(fontSize = 28.sp, lineHeight = 36.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.5).sp),
    headlineMedium = Base.copy(fontSize = 24.sp, lineHeight = 32.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.4).sp),
    headlineSmall = Base.copy(fontSize = 20.sp, lineHeight = 28.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.3).sp),
    titleLarge = Base.copy(fontSize = 18.sp, lineHeight = 26.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.2).sp),
    titleMedium = Base.copy(fontSize = 16.sp, lineHeight = 24.sp, fontWeight = FontWeight.SemiBold),
    titleSmall = Base.copy(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.SemiBold),
    bodyLarge = Base.copy(fontSize = 16.sp, lineHeight = 24.sp),
    bodyMedium = Base.copy(fontSize = 14.sp, lineHeight = 20.sp),
    bodySmall = Base.copy(fontSize = 12.sp, lineHeight = 16.sp),
    labelLarge = Base.copy(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium),
    labelMedium = Base.copy(fontSize = 12.sp, lineHeight = 16.sp, fontWeight = FontWeight.Medium),
    labelSmall = Base.copy(fontSize = 11.sp, lineHeight = 16.sp, fontWeight = FontWeight.Medium),
)
