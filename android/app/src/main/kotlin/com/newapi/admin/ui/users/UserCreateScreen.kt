package com.newapi.admin.ui.users

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Roles
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppSegmentedTabs
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.rememberCurrentUser
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import kotlinx.coroutines.flow.first

/** New user form: username, password, optional display name, and (for root) the admin role. */
@Composable
fun UserCreateScreen(onBack: () -> Unit, onCreated: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { UserCreateViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val me = rememberCurrentUser()
    var passwordVisible by rememberSaveable { mutableStateOf(false) }
    val enabled = !state.isSubmitting && !state.created
    val transformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation()

    // Leaving mid-request would cancel it client-side while the server may still create the user, and the
    // list would never hear of it; stay until the answer arrives.
    BackHandler(enabled = state.isSubmitting) {}

    // A pop is ignored unless this entry is RESUMED (for example while the app is paused), so wait for it.
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val currentOnCreated by rememberUpdatedState(onCreated)
    LaunchedEffect(state.created) {
        if (!state.created) return@LaunchedEffect
        lifecycle.currentStateFlow.first { it.isAtLeast(Lifecycle.State.RESUMED) }
        currentOnCreated()
    }

    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.users_add),
                navigationIcon = {
                    IconButton(onClick = onBack, enabled = !state.isSubmitting) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .consumeWindowInsets(padding)
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            AppCard(Modifier.fillMaxWidth()) {
                Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                    val usernameError = state.fieldErrors[CreateField.USERNAME]
                    AppTextField(
                        value = viewModel.username,
                        onValueChange = viewModel::onUsernameChange,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = enabled,
                        singleLine = true,
                        label = stringResource(R.string.users_field_username),
                        isError = usernameError != null,
                        supportingText = usernameError?.text() ?: stringResource(R.string.users_create_username_hint),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Ascii, imeAction = ImeAction.Next),
                    )

                    val passwordError = state.fieldErrors[CreateField.PASSWORD]
                    AppTextField(
                        value = viewModel.password,
                        onValueChange = viewModel::onPasswordChange,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = enabled,
                        singleLine = true,
                        label = stringResource(R.string.users_field_password),
                        trailing = {
                            IconButton(onClick = { passwordVisible = !passwordVisible }) {
                                Icon(
                                    if (passwordVisible) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility,
                                    contentDescription = stringResource(if (passwordVisible) R.string.users_hide_password else R.string.users_show_password),
                                    modifier = Modifier.size(18.dp),
                                )
                            }
                        },
                        isError = passwordError != null,
                        supportingText = passwordError?.text() ?: stringResource(R.string.users_password_length),
                        visualTransformation = transformation,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                    )

                    val confirmError = state.fieldErrors[CreateField.CONFIRM]
                    AppTextField(
                        value = viewModel.confirmPassword,
                        onValueChange = viewModel::onConfirmPasswordChange,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = enabled,
                        singleLine = true,
                        label = stringResource(R.string.users_password_confirm),
                        isError = confirmError != null,
                        supportingText = confirmError?.text(),
                        visualTransformation = transformation,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                    )

                    val displayNameError = state.fieldErrors[CreateField.DISPLAY_NAME]
                    AppTextField(
                        value = viewModel.displayName,
                        onValueChange = viewModel::onDisplayNameChange,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = enabled,
                        singleLine = true,
                        label = stringResource(R.string.users_field_display_name),
                        placeholder = viewModel.username.trim().takeIf { it.isNotEmpty() },
                        isError = displayNameError != null,
                        supportingText = displayNameError?.text() ?: stringResource(R.string.users_create_display_name_hint),
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = { viewModel.submit() }),
                    )

                    // Only root may create administrators; everyone else always creates regular users.
                    if (me?.isRoot == true) {
                        val roles = listOf(Roles.COMMON, Roles.ADMIN)
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(stringResource(R.string.users_field_role), style = MaterialTheme.typography.labelLarge)
                            AppSegmentedTabs(
                                options = roles.map { option ->
                                    stringResource(if (option == Roles.ADMIN) R.string.users_create_role_admin else R.string.users_create_role_user)
                                },
                                selectedIndex = roles.indexOf(viewModel.role),
                                onSelect = { viewModel.onRoleChange(roles[it]) },
                                modifier = Modifier.fillMaxWidth(),
                                enabled = enabled,
                            )
                        }
                    }
                }
            }

            Text(
                stringResource(R.string.users_create_note),
                style = MaterialTheme.typography.bodySmall,
                color = AppTheme.colors.mutedForeground,
                modifier = Modifier.padding(horizontal = 4.dp),
            )

            state.error?.let { error ->
                AppAlert(text = errorMessage(error), tone = Tone.Danger, icon = Icons.Outlined.ErrorOutline)
            }

            AppButton(
                stringResource(R.string.users_create_submit),
                onClick = viewModel::submit,
                modifier = Modifier.fillMaxWidth(),
                size = ButtonSize.Large,
                enabled = enabled,
                loading = state.isSubmitting,
            )
        }
    }
}
