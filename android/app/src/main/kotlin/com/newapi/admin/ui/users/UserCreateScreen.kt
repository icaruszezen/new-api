package com.newapi.admin.ui.users

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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Roles
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.rememberCurrentUser

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

    LaunchedEffect(state.created) {
        if (state.created) onCreated()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.users_add)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_back))
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
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            val usernameError = state.fieldErrors[CreateField.USERNAME]
            OutlinedTextField(
                value = viewModel.username,
                onValueChange = viewModel::onUsernameChange,
                modifier = Modifier.fillMaxWidth(),
                enabled = enabled,
                singleLine = true,
                label = { Text(stringResource(R.string.users_field_username)) },
                isError = usernameError != null,
                supportingText = { Text(usernameError?.text() ?: stringResource(R.string.users_create_username_hint)) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Ascii, imeAction = ImeAction.Next),
            )

            val passwordError = state.fieldErrors[CreateField.PASSWORD]
            OutlinedTextField(
                value = viewModel.password,
                onValueChange = viewModel::onPasswordChange,
                modifier = Modifier.fillMaxWidth(),
                enabled = enabled,
                singleLine = true,
                label = { Text(stringResource(R.string.users_field_password)) },
                trailingIcon = {
                    IconButton(onClick = { passwordVisible = !passwordVisible }) {
                        Icon(
                            if (passwordVisible) Icons.Filled.VisibilityOff else Icons.Filled.Visibility,
                            contentDescription = stringResource(if (passwordVisible) R.string.users_hide_password else R.string.users_show_password),
                        )
                    }
                },
                isError = passwordError != null,
                supportingText = { Text(passwordError?.text() ?: stringResource(R.string.users_password_length)) },
                visualTransformation = transformation,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
            )

            val confirmError = state.fieldErrors[CreateField.CONFIRM]
            OutlinedTextField(
                value = viewModel.confirmPassword,
                onValueChange = viewModel::onConfirmPasswordChange,
                modifier = Modifier.fillMaxWidth(),
                enabled = enabled,
                singleLine = true,
                label = { Text(stringResource(R.string.users_password_confirm)) },
                isError = confirmError != null,
                supportingText = confirmError?.let { { Text(it.text()) } },
                visualTransformation = transformation,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
            )

            val displayNameError = state.fieldErrors[CreateField.DISPLAY_NAME]
            OutlinedTextField(
                value = viewModel.displayName,
                onValueChange = viewModel::onDisplayNameChange,
                modifier = Modifier.fillMaxWidth(),
                enabled = enabled,
                singleLine = true,
                label = { Text(stringResource(R.string.users_field_display_name)) },
                placeholder = viewModel.username.trim().takeIf { it.isNotEmpty() }?.let { { Text(it) } },
                isError = displayNameError != null,
                supportingText = { Text(displayNameError?.text() ?: stringResource(R.string.users_create_display_name_hint)) },
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { viewModel.submit() }),
            )

            // Only root may create administrators; everyone else always creates regular users.
            if (me?.isRoot == true) {
                Text(
                    stringResource(R.string.users_field_role),
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.padding(top = 4.dp),
                )
                val roles = listOf(Roles.COMMON, Roles.ADMIN)
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    roles.forEachIndexed { index, option ->
                        SegmentedButton(
                            selected = viewModel.role == option,
                            onClick = { viewModel.onRoleChange(option) },
                            shape = SegmentedButtonDefaults.itemShape(index, roles.size),
                            enabled = enabled,
                        ) {
                            Text(stringResource(if (option == Roles.ADMIN) R.string.users_create_role_admin else R.string.users_create_role_user))
                        }
                    }
                }
            }

            Text(
                stringResource(R.string.users_create_note),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(vertical = 8.dp),
            )

            state.error?.let { error ->
                Text(errorMessage(error), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error)
            }

            Button(onClick = viewModel::submit, enabled = enabled, modifier = Modifier.fillMaxWidth()) {
                if (state.isSubmitting) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                } else {
                    Text(stringResource(R.string.users_create_submit))
                }
            }
        }
    }
}
