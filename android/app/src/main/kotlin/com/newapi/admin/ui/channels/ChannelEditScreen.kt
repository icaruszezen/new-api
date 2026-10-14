package com.newapi.admin.ui.channels

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.CloudDownload
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.UnfoldMore
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.ChannelType
import com.newapi.admin.core.model.ChannelTypes
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppDialog
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppRemovableChip
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppSwitch
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.SearchField
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import com.newapi.admin.ui.theme.Tone

/** Model chips previewed under the models field. */
private const val MODEL_PREVIEW_COUNT = 40

/** Key formats the web prompts for (`TYPE_TO_KEY_PROMPT`). */
private val keyFormats: Map<Int, String> = mapOf(
    15 to "APIKey|SecretKey",
    18 to "APPID|APISecret|APIKey",
    22 to "APIKey-AppId",
    23 to "AppId|SecretId|SecretKey",
    33 to "Ak|Sk|Region",
    50 to "AccessKey|SecretKey",
    51 to "AccessKeyID|SecretAccessKey",
    57 to "{\"access_token\": \"…\", \"account_id\": \"…\"}",
)

/** Types the server rejects without settings this form does not edit (Vertex region JSON, advanced custom config). */
private val typesNeedingWebSettings: Set<Int> = setOf(41, 58)

/** Creates a channel when [channelId] is null. */
@Composable
fun ChannelEditScreen(channelId: Int?, onBack: () -> Unit, onSaved: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "channel-edit-${channelId ?: 0}") { ChannelEditViewModel(container, channelId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val groups by viewModel.groups.collectAsStateWithLifecycle()
    val upstream by viewModel.upstream.collectAsStateWithLifecycle()
    val permissions = rememberChannelPermissions()
    val canEdit = if (viewModel.isCreate) permissions.sensitiveWrite else permissions.write
    val canEditSensitive = canEdit && permissions.sensitiveWrite
    val snackbarHostState = remember { SnackbarHostState() }
    var confirmDiscard by rememberSaveable { mutableStateOf(false) }
    var showTypePicker by rememberSaveable { mutableStateOf(false) }

    val guardBack = viewModel.isDirty && !state.saved
    val requestBack: () -> Unit = {
        if (guardBack) {
            confirmDiscard = true
        } else {
            onBack()
        }
    }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    // Leaving during a save would cancel it before the change is announced, although the server may apply it.
    BackHandler(enabled = guardBack || state.isSaving) {
        if (!state.isSaving) confirmDiscard = true
    }
    LaunchedEffect(state.saved) {
        if (state.saved) onSaved()
    }

    val formReady = canEdit && !state.isLoading && state.loadError == null
    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(if (viewModel.isCreate) R.string.channels_create_title else R.string.channels_edit_title),
                navigationIcon = {
                    IconButton(onClick = requestBack, enabled = !state.isSaving) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
                actions = {
                    if (formReady) {
                        if (state.isSaving) {
                            CircularProgressIndicator(
                                Modifier.padding(horizontal = 24.dp).size(20.dp),
                                strokeWidth = 2.dp,
                                color = AppTheme.colors.mutedForeground,
                            )
                        } else {
                            AppButton(
                                stringResource(R.string.common_save),
                                onClick = { viewModel.save(canEditSensitive) },
                                // With the bar's own 4dp, the button lines up with the 16dp content gutter.
                                modifier = Modifier.padding(end = 12.dp),
                                size = ButtonSize.Small,
                                enabled = !state.saved && (viewModel.isCreate || viewModel.isDirty),
                            )
                        }
                    }
                },
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        val loadError = state.loadError
        val contentModifier = Modifier.fillMaxSize().padding(padding)
        when {
            !canEdit -> EmptyState(
                text = stringResource(if (viewModel.isCreate) R.string.channels_no_create_permission else R.string.channels_no_write_permission),
                icon = Icons.Outlined.Lock,
                modifier = contentModifier,
            )
            state.isLoading -> FullScreenLoading(contentModifier)
            loadError != null -> FullScreenError(loadError, onRetry = viewModel::load, modifier = contentModifier)
            else -> ChannelFormFields(
                viewModel = viewModel,
                state = state,
                groups = groups,
                isFetchingModels = upstream.isLoading,
                canEditSensitive = canEditSensitive,
                canFetchModels = !viewModel.isCreate && permissions.operate,
                onPickType = { showTypePicker = true },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .consumeWindowInsets(padding)
                    .imePadding(),
            )
        }
    }

    if (showTypePicker) {
        ChannelTypeSheet(
            selected = viewModel.form.type,
            onSelect = { type ->
                showTypePicker = false
                viewModel.update { it.copy(type = type) }
            },
            onDismiss = { showTypePicker = false },
        )
    }

    if (upstream.models != null) {
        UpstreamModelsDialog(
            picker = upstream,
            currentModels = viewModel.form.modelList,
            onToggle = viewModel::toggleUpstreamModel,
            onSetSelection = viewModel::setUpstreamSelection,
            onApply = viewModel::applyUpstreamModels,
            onDismiss = viewModel::dismissUpstreamModels,
        )
    }

    if (confirmDiscard) {
        ConfirmDialog(
            title = stringResource(R.string.channels_discard_title),
            text = stringResource(R.string.channels_discard_message),
            confirmText = stringResource(R.string.channels_discard_confirm),
            destructive = true,
            onConfirm = {
                confirmDiscard = false
                onBack()
            },
            onDismiss = { confirmDiscard = false },
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChannelFormFields(
    viewModel: ChannelEditViewModel,
    state: ChannelEditState,
    groups: GroupOptions,
    isFetchingModels: Boolean,
    canEditSensitive: Boolean,
    canFetchModels: Boolean,
    onPickType: () -> Unit,
    modifier: Modifier,
) {
    val form = viewModel.form
    val errors = if (state.showErrors) viewModel.errors(canEditSensitive) else ChannelFormErrors()
    val editable = !state.isSaving
    val sensitiveEditable = editable && canEditSensitive
    val sensitiveHint = if (canEditSensitive) null else stringResource(R.string.channels_sensitive_hint)

    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        // Connection: type, name, key and base URL.
        FormCard {
            // Type: a read-only field that opens the searchable picker.
            Box {
                AppTextField(
                    value = "${channelTypeLabel(form.type)} (#${form.type})",
                    onValueChange = {},
                    modifier = Modifier.fillMaxWidth(),
                    readOnly = true,
                    enabled = sensitiveEditable,
                    label = stringResource(R.string.channels_field_type),
                    trailing = { Icon(Icons.Outlined.UnfoldMore, contentDescription = null, modifier = Modifier.size(18.dp)) },
                    supportingText = sensitiveHint,
                )
                if (sensitiveEditable) {
                    Box(Modifier.matchParentSize().clickable(onClick = onPickType))
                }
            }
            if (form.type in typesNeedingWebSettings && viewModel.typeIsNew) {
                AppAlert(
                    text = stringResource(R.string.channels_type_needs_web),
                    tone = Tone.Danger,
                    icon = Icons.Outlined.ErrorOutline,
                )
            }

            FormField(
                value = form.name,
                onValueChange = { value -> viewModel.update { it.copy(name = value) } },
                label = stringResource(R.string.channels_field_name),
                enabled = editable,
                error = errors.name,
            )

            val keyHint = when {
                sensitiveHint != null -> sensitiveHint
                state.multiKeySize > 0 -> stringResource(R.string.channels_key_multi_hint, state.multiKeySize)
                else -> keyFormats[form.type]?.let { stringResource(R.string.channels_key_format, it) }
            }
            FormField(
                value = form.key,
                onValueChange = { value -> viewModel.update { it.copy(key = value) } },
                label = stringResource(R.string.channels_field_key),
                enabled = sensitiveEditable,
                error = errors.key,
                hint = keyHint,
                placeholder = stringResource(
                    when {
                        !viewModel.isCreate -> R.string.channels_key_placeholder_edit
                        form.batch -> R.string.channels_key_placeholder_batch
                        else -> R.string.channels_key_placeholder_create
                    },
                ),
                singleLine = false,
                minLines = if (form.batch) 3 else 1,
                maxLines = 8,
                monospace = true,
            )
            if (viewModel.isCreate) {
                SwitchRow(
                    label = stringResource(R.string.channels_batch_create),
                    hint = stringResource(R.string.channels_batch_hint),
                    checked = form.batch,
                    enabled = editable,
                    onCheckedChange = { value -> viewModel.update { it.copy(batch = value) } },
                )
            }

            FormField(
                value = form.baseUrl,
                onValueChange = { value -> viewModel.update { it.copy(baseUrl = value) } },
                label = stringResource(R.string.channels_field_base_url),
                enabled = sensitiveEditable,
                error = errors.baseUrl,
                hint = sensitiveHint ?: stringResource(
                    if (form.type in ChannelTypes.baseUrlRequired) R.string.channels_base_url_hint_required else R.string.channels_base_url_hint_optional,
                ),
                keyboardType = KeyboardType.Uri,
            )
        }

        // Models: the list, its preview chips, upstream fetch, test model and mapping.
        FormCard {
            // The chips and the fetch button belong to the models field, so they sit closer to it.
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                FormField(
                    value = form.models,
                    onValueChange = { value -> viewModel.update { it.copy(models = value) } },
                    label = stringResource(R.string.channels_field_models),
                    enabled = editable,
                    error = errors.models,
                    hint = stringResource(R.string.channels_models_hint),
                    singleLine = false,
                    maxLines = 6,
                    monospace = true,
                )
                val models = form.modelList
                if (models.isNotEmpty()) {
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                        models.take(MODEL_PREVIEW_COUNT).forEach { model ->
                            AppRemovableChip(
                                label = model,
                                onRemove = { viewModel.removeModel(model) },
                                removeDescription = stringResource(R.string.channels_remove_model, model),
                                enabled = editable,
                            )
                        }
                        if (models.size > MODEL_PREVIEW_COUNT) {
                            Text(
                                stringResource(R.string.channels_models_preview_more, models.size - MODEL_PREVIEW_COUNT),
                                style = MaterialTheme.typography.bodySmall,
                                color = AppTheme.colors.mutedForeground,
                                modifier = Modifier.align(Alignment.CenterVertically),
                            )
                        }
                    }
                }
                if (canFetchModels) {
                    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        // loading = isFetchingModels also keeps the button disabled while the fetch runs.
                        AppButton(
                            stringResource(R.string.channels_fetch_models),
                            onClick = viewModel::fetchUpstreamModels,
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                            enabled = editable,
                            loading = isFetchingModels,
                            leadingIcon = Icons.Outlined.CloudDownload,
                        )
                        Text(
                            stringResource(R.string.channels_fetch_models_hint),
                            style = MaterialTheme.typography.bodySmall,
                            color = AppTheme.colors.mutedForeground,
                        )
                    }
                }
            }
            FormField(
                value = form.testModel,
                onValueChange = { value -> viewModel.update { it.copy(testModel = value) } },
                label = stringResource(R.string.channels_field_test_model),
                enabled = editable,
                hint = stringResource(R.string.channels_test_model_hint),
                monospace = true,
            )
            FormField(
                value = form.modelMapping,
                onValueChange = { value -> viewModel.update { it.copy(modelMapping = value) } },
                label = stringResource(R.string.channels_field_model_mapping),
                enabled = editable,
                error = errors.modelMapping,
                hint = stringResource(R.string.channels_model_mapping_hint),
                placeholder = "{\"gpt-4o\": \"gpt-4o-2024-08-06\"}",
                singleLine = false,
                minLines = 3,
                maxLines = 10,
                keyboardType = KeyboardType.Ascii,
                monospace = true,
            )
        }

        // Routing: groups, priority and weight, then the bookkeeping fields.
        FormCard {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    stringResource(R.string.channels_field_groups),
                    style = MaterialTheme.typography.labelLarge,
                    color = if (errors.groups != null) AppTheme.colors.destructive else AppTheme.colors.foreground,
                )
                // Selected groups the server no longer lists stay visible so they can be removed.
                val options = form.groups.filter { it !in groups.names } + groups.names
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    options.forEach { group ->
                        val selected = group in form.groups
                        AppFilterChip(
                            selected = selected,
                            onClick = { viewModel.toggleGroup(group) },
                            label = group,
                            enabled = editable,
                            leadingIcon = if (selected) Icons.Outlined.Check else null,
                        )
                    }
                }
                when {
                    groups.isLoading -> CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = AppTheme.colors.mutedForeground)
                    groups.failed -> Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            stringResource(R.string.channels_groups_load_failed),
                            style = MaterialTheme.typography.bodySmall,
                            color = AppTheme.colors.destructive,
                            modifier = Modifier.weight(1f),
                        )
                        AppButton(
                            stringResource(R.string.common_retry),
                            onClick = viewModel::loadGroups,
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                        )
                    }
                }
                errors.groups?.let { error ->
                    Text(stringResource(error), style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.destructive)
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                FormField(
                    value = form.priority,
                    onValueChange = { value -> viewModel.update { it.copy(priority = value) } },
                    label = stringResource(R.string.channels_field_priority),
                    enabled = editable,
                    error = errors.priority,
                    keyboardType = KeyboardType.Number,
                    modifier = Modifier.weight(1f),
                )
                FormField(
                    value = form.weight,
                    onValueChange = { value -> viewModel.update { it.copy(weight = value) } },
                    label = stringResource(R.string.channels_field_weight),
                    enabled = editable,
                    error = errors.weight,
                    keyboardType = KeyboardType.Number,
                    modifier = Modifier.weight(1f),
                )
            }

            FormField(
                value = form.tag,
                onValueChange = { value -> viewModel.update { it.copy(tag = value) } },
                label = stringResource(R.string.channels_field_tag),
                enabled = editable,
            )
            FormField(
                value = form.remark,
                onValueChange = { value -> viewModel.update { it.copy(remark = value) } },
                label = stringResource(R.string.channels_field_remark),
                enabled = editable,
                error = errors.remark,
                singleLine = false,
                maxLines = 4,
                autoCorrect = true,
            )
            SwitchRow(
                label = stringResource(R.string.channels_field_auto_ban),
                hint = stringResource(R.string.channels_auto_ban_hint),
                checked = form.autoBan,
                enabled = editable,
                onCheckedChange = { value -> viewModel.update { it.copy(autoBan = value) } },
            )
        }
    }
}

/** One group of form fields: a bordered card with roomy spacing. */
@Composable
private fun FormCard(content: @Composable () -> Unit) {
    AppCard(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) { content() }
    }
}

@Composable
private fun FormField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    error: Int? = null,
    hint: String? = null,
    placeholder: String? = null,
    singleLine: Boolean = true,
    minLines: Int = 1,
    maxLines: Int = if (singleLine) 1 else 6,
    keyboardType: KeyboardType = KeyboardType.Text,
    /** Keys, model names, JSON. */
    monospace: Boolean = false,
    /** Off by default: most fields hold identifiers, keys or URLs. */
    autoCorrect: Boolean = false,
) {
    val supporting = error?.let { stringResource(it) } ?: hint
    AppTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = modifier.fillMaxWidth(),
        label = label,
        placeholder = placeholder,
        supportingText = supporting,
        isError = error != null,
        enabled = enabled,
        singleLine = singleLine,
        minLines = minLines,
        maxLines = maxLines,
        textStyle = if (monospace) MaterialTheme.typography.bodyMedium.copy(fontFamily = GeistMono) else MaterialTheme.typography.bodyLarge,
        keyboardOptions = KeyboardOptions(autoCorrectEnabled = autoCorrect, keyboardType = keyboardType),
    )
}

@Composable
private fun SwitchRow(label: String, hint: String?, checked: Boolean, enabled: Boolean, onCheckedChange: (Boolean) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.medium)
            .toggleable(value = checked, enabled = enabled, role = Role.Switch, onValueChange = onCheckedChange)
            .border(1.dp, AppTheme.colors.border, MaterialTheme.shapes.medium)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(label, style = MaterialTheme.typography.labelLarge)
            if (hint != null) {
                Text(hint, style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground)
            }
        }
        Spacer(Modifier.width(12.dp))
        AppSwitch(checked = checked, onCheckedChange = null, enabled = enabled)
    }
}

@Composable
private fun ChannelTypeSheet(selected: Int, onSelect: (Int) -> Unit, onDismiss: () -> Unit) {
    var search by rememberSaveable { mutableStateOf("") }
    // A channel of a type the picker hides keeps its type selectable.
    val selectableIds = ChannelTypes.selectable.map { it.id }
    val optionIds = if (selected in selectableIds) selectableIds else listOf(selected) + selectableIds
    val options = optionIds.map { ChannelType(it, channelTypeLabel(it)) }
    val keyword = search.trim()
    val shown = if (keyword.isEmpty()) {
        options
    } else {
        // The English name also matches, so "baidu" finds 百度.
        options.filter {
            it.name.contains(keyword, ignoreCase = true) ||
                ChannelTypes.nameOf(it.id).contains(keyword, ignoreCase = true) ||
                it.id.toString() == keyword
        }
    }
    AppBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        Text(
            stringResource(R.string.channels_type_picker_title),
            style = MaterialTheme.typography.titleLarge,
            modifier = Modifier.padding(horizontal = 20.dp),
        )
        SearchField(
            query = search,
            onQueryChange = { search = it },
            onSearch = {},
            placeholder = stringResource(R.string.channels_type_search),
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
        )
        LazyColumn(Modifier.fillMaxWidth(), contentPadding = PaddingValues(bottom = 16.dp)) {
            items(shown, key = { it.id }) { type ->
                val isSelected = type.id == selected
                ListItem(
                    headlineContent = { Text(type.name, style = MaterialTheme.typography.bodyMedium) },
                    supportingContent = {
                        Text("#${type.id}", style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground)
                    },
                    trailingContent = {
                        if (isSelected) Icon(Icons.Outlined.Check, contentDescription = null, tint = AppTheme.colors.foreground, modifier = Modifier.size(18.dp))
                    },
                    colors = ListItemDefaults.colors(containerColor = if (isSelected) AppTheme.colors.accent else Color.Transparent),
                    modifier = Modifier
                        .padding(horizontal = 8.dp)
                        .clip(MaterialTheme.shapes.small)
                        .clickable { onSelect(type.id) },
                )
            }
        }
    }
}

@Composable
private fun UpstreamModelsDialog(
    picker: UpstreamModels,
    currentModels: List<String>,
    onToggle: (String) -> Unit,
    onSetSelection: (models: List<String>, selected: Boolean) -> Unit,
    onApply: () -> Unit,
    onDismiss: () -> Unit,
) {
    val models = picker.models ?: return
    var filter by rememberSaveable { mutableStateOf("") }
    val keyword = filter.trim()
    val shown = if (keyword.isEmpty()) models else models.filter { it.contains(keyword, ignoreCase = true) }
    val current = remember(currentModels) { currentModels.toSet() }

    AppDialog(
        onDismissRequest = onDismiss,
        title = stringResource(R.string.channels_fetch_models_title),
        description = stringResource(R.string.channels_selected_count, picker.selected.size, models.size),
        confirmButton = { AppButton(stringResource(R.string.common_apply), onClick = onApply) },
        dismissButton = { AppButton(stringResource(R.string.common_cancel), onClick = onDismiss, variant = ButtonVariant.Outline) },
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            AppTextField(
                value = filter,
                onValueChange = { filter = it },
                modifier = Modifier.fillMaxWidth(),
                placeholder = stringResource(R.string.channels_test_model_filter),
                leadingIcon = Icons.Outlined.Search,
                singleLine = true,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                AppButton(
                    stringResource(R.string.channels_select_all),
                    onClick = { onSetSelection(shown, true) },
                    variant = ButtonVariant.Outline,
                    size = ButtonSize.Small,
                )
                AppButton(
                    stringResource(R.string.channels_select_none),
                    onClick = { onSetSelection(shown, false) },
                    variant = ButtonVariant.Outline,
                    size = ButtonSize.Small,
                )
            }
            Column(
                Modifier
                    .border(1.dp, AppTheme.colors.border, MaterialTheme.shapes.medium)
                    .heightIn(max = 300.dp)
                    .verticalScroll(rememberScrollState())
                    .padding(vertical = 4.dp),
            ) {
                shown.forEach { model ->
                    val checked = model in picker.selected
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .toggleable(value = checked, role = Role.Checkbox, onValueChange = { onToggle(model) })
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        CheckboxIndicator(checked)
                        Spacer(Modifier.width(12.dp))
                        Text(
                            model,
                            style = MaterialTheme.typography.bodySmall,
                            fontFamily = GeistMono,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f),
                        )
                        if (model !in current) {
                            Spacer(Modifier.width(8.dp))
                            StatusPill(stringResource(R.string.channels_model_new), Tone.Info)
                        }
                    }
                }
            }
        }
    }
}

/** shadcn checkbox: a small rounded box, filled with a check when on. Draw-only; the row owns the click. */
@Composable
private fun CheckboxIndicator(checked: Boolean) {
    val colors = AppTheme.colors
    val shape = RoundedCornerShape(4.dp)
    Box(
        Modifier
            .size(16.dp)
            .then(if (checked) Modifier.background(colors.primary, shape) else Modifier.border(1.dp, colors.input, shape)),
        contentAlignment = Alignment.Center,
    ) {
        if (checked) Icon(Icons.Outlined.Check, contentDescription = null, tint = colors.primaryForeground, modifier = Modifier.size(12.dp))
    }
}
