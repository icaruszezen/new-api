package com.newapi.admin.ui.users

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.newapi.admin.R
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.Roles
import com.newapi.admin.core.model.UserStatus
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors

/** Server limits from `model.User` validator tags, counted in runes like Go's validator. */
internal const val MAX_NAME_LENGTH = 20
internal const val MAX_REMARK_LENGTH = 255
internal val PASSWORD_LENGTH = 8..20

/** `common.MaxWalletQuota` (2^53-1): the largest balance `/api/user/manage add_quota` accepts. */
internal const val MAX_WALLET_QUOTA = 9_007_199_254_740_991L

/** `PUT /api/user/` (and the own-password update) silently treat this exact password as "unchanged". */
internal const val UNCHANGED_PASSWORD_SENTINEL = "\$I_LOVE_U"

/** Status filter value for soft-deleted users (`/api/user/search?status=-1`). */
internal const val STATUS_DELETED = -1

internal fun String.runeCount(): Int = codePointCount(0, length)

/** String resource describing why an admin-set password is rejected, or null when it is acceptable. */
internal fun passwordProblem(password: String): Int? = when {
    password.isEmpty() -> R.string.common_required
    password.runeCount() !in PASSWORD_LENGTH -> R.string.users_password_length_error
    password == UNCHANGED_PASSWORD_SENTINEL -> R.string.users_password_reserved
    else -> null
}

/** `GET /api/group/` for the group pickers; a failure leaves the list empty with [failed] set. */
data class GroupList(
    val names: List<String> = emptyList(),
    val isLoading: Boolean = false,
    val failed: Boolean = false,
)

@Composable
internal fun roleLabel(role: Int): String = when (role) {
    Roles.GUEST -> stringResource(R.string.users_role_guest)
    Roles.COMMON -> stringResource(R.string.users_role_user)
    Roles.ADMIN -> stringResource(R.string.users_role_admin)
    Roles.ROOT -> stringResource(R.string.users_role_root)
    else -> stringResource(R.string.users_role_other, role)
}

@Composable
internal fun RolePill(role: Int, modifier: Modifier = Modifier) {
    val tone = when {
        role >= Roles.ROOT -> Tone.Warning
        role >= Roles.ADMIN -> Tone.Info
        else -> Tone.Neutral
    }
    StatusPill(roleLabel(role), tone, modifier)
}

@Composable
internal fun userStatusLabel(user: AdminUser): String = when {
    user.isDeleted -> stringResource(R.string.users_status_deleted)
    user.status == UserStatus.ENABLED -> stringResource(R.string.common_enabled)
    user.status == UserStatus.DISABLED -> stringResource(R.string.common_disabled)
    else -> stringResource(R.string.users_status_other, user.status)
}

@Composable
internal fun UserStatusPill(user: AdminUser, modifier: Modifier = Modifier) {
    val tone = when {
        user.isDeleted -> Tone.Neutral
        user.status == UserStatus.ENABLED -> Tone.Success
        user.status == UserStatus.DISABLED -> Tone.Danger
        else -> Tone.Neutral
    }
    StatusPill(userStatusLabel(user), tone, modifier)
}

/**
 * Thin remaining / (remaining + used) bar like the web quota cell: rose at 10 % or less, amber at
 * 30 % or less. A negative balance (possible after a subtract) counts as nothing left.
 */
@Composable
internal fun QuotaBar(quota: Long, usedQuota: Long, modifier: Modifier = Modifier) {
    val total = quota.toDouble() + usedQuota.toDouble()
    val fraction = if (total > 0) (quota.coerceAtLeast(0) / total).coerceIn(0.0, 1.0).toFloat() else 0f
    val color = when {
        fraction <= 0.1f -> Tone.Danger.colors().content
        fraction <= 0.3f -> Tone.Warning.colors().content
        else -> MaterialTheme.colorScheme.primary
    }
    LinearProgressIndicator(
        progress = { fraction },
        modifier = modifier.fillMaxWidth().height(4.dp),
        color = color,
        trackColor = MaterialTheme.colorScheme.surfaceContainerHighest,
        strokeCap = StrokeCap.Round,
        gapSize = 0.dp,
        drawStopIndicator = {},
    )
}

/**
 * Read-only dropdown over [options]. With [allLabel] an extra first entry selects "" (no group).
 * A [value] missing from [options] (the server does not validate stored groups) stays selectable.
 */
@Composable
internal fun GroupDropdown(
    value: String,
    options: List<String>,
    onSelect: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    allLabel: String? = null,
    enabled: Boolean = true,
    supportingText: String? = null,
) {
    var expanded by rememberSaveable { mutableStateOf(false) }
    val entries = (options + value).filter { it.isNotEmpty() }.distinct().sorted()
    ExposedDropdownMenuBox(
        expanded = expanded && enabled,
        onExpandedChange = { if (enabled) expanded = it },
        modifier = modifier,
    ) {
        OutlinedTextField(
            value = value.ifEmpty { allLabel.orEmpty() },
            onValueChange = {},
            readOnly = true,
            enabled = enabled,
            singleLine = true,
            label = { Text(label) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded && enabled) },
            supportingText = supportingText?.let { { Text(it) } },
            modifier = Modifier.fillMaxWidth().menuAnchor(MenuAnchorType.PrimaryNotEditable, enabled),
        )
        ExposedDropdownMenu(expanded = expanded && enabled, onDismissRequest = { expanded = false }) {
            if (allLabel != null) {
                DropdownMenuItem(
                    text = { Text(allLabel) },
                    onClick = {
                        onSelect("")
                        expanded = false
                    },
                    contentPadding = ExposedDropdownMenuDefaults.ItemContentPadding,
                )
            }
            entries.forEach { group ->
                DropdownMenuItem(
                    text = { Text(group) },
                    onClick = {
                        onSelect(group)
                        expanded = false
                    },
                    contentPadding = ExposedDropdownMenuDefaults.ItemContentPadding,
                )
            }
        }
    }
}
