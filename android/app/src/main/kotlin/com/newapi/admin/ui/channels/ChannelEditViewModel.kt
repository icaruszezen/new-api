package com.newapi.admin.ui.channels

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelDraft
import com.newapi.admin.core.model.ChannelPatch
import com.newapi.admin.core.model.ChannelTypes
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.coroutines.channels.Channel as MessageChannel

/** Channel type the server requires base_url for (`validateChannel`). */
private const val TYPE_NEW_API = 60

/** Channel type whose key `validateChannel` checks as OAuth JSON, but only when the body carries the type. */
private const val TYPE_CODEX = 57

/** `group` is a varchar(64) column. */
private const val MAX_GROUP_LENGTH = 64

/** `remark` is a varchar(255) column. */
private const val MAX_REMARK_LENGTH = 255

/** Editable channel fields as typed. Numbers stay text so partial input survives. */
data class ChannelForm(
    val type: Int = 1,
    val name: String = "",
    /** Create: required. Edit: blank keeps the current key. */
    val key: String = "",
    /** Create only: one channel per key line. */
    val batch: Boolean = false,
    val baseUrl: String = "",
    /** Comma separated. */
    val models: String = "",
    val groups: List<String> = listOf("default"),
    val priority: String = "",
    val weight: String = "",
    val tag: String = "",
    val remark: String = "",
    val testModel: String = "",
    val modelMapping: String = "",
    val autoBan: Boolean = true,
) {
    val modelList: List<String> get() = Channel.splitCsv(models).distinct()

    companion object {
        fun of(channel: Channel) = ChannelForm(
            type = channel.type,
            name = channel.name,
            baseUrl = channel.baseUrl.orEmpty(),
            models = channel.modelList.joinToString(","),
            groups = channel.groupList.ifEmpty { listOf("default") },
            priority = channel.priority?.toString().orEmpty(),
            weight = channel.weight?.toString().orEmpty(),
            tag = channel.tag.orEmpty(),
            remark = channel.remark.orEmpty(),
            testModel = channel.testModel.orEmpty(),
            modelMapping = channel.modelMapping.orEmpty(),
            // The server treats a missing auto_ban as off.
            autoBan = channel.autoBan == 1,
        )
    }
}

/** String resource ids of field errors; null means valid. */
data class ChannelFormErrors(
    val name: Int? = null,
    val key: Int? = null,
    val baseUrl: Int? = null,
    val models: Int? = null,
    val groups: Int? = null,
    val priority: Int? = null,
    val weight: Int? = null,
    val remark: Int? = null,
    val modelMapping: Int? = null,
) {
    val isValid: Boolean get() = this == ChannelFormErrors()
}

data class ChannelEditState(
    /** Edit only: loading the channel. */
    val isLoading: Boolean = false,
    val loadError: Throwable? = null,
    /** Edit only: key count of a multi-key channel (a new key replaces them all), 0 otherwise. */
    val multiKeySize: Int = 0,
    val isSaving: Boolean = false,
    /** Field errors are shown after the first save attempt. */
    val showErrors: Boolean = false,
    /** Saved; the screen leaves. */
    val saved: Boolean = false,
)

/** Upstream model picker: the fetched list and the ones the admin keeps checked. */
data class UpstreamModels(
    val isLoading: Boolean = false,
    /** Non-null while the picker is open. */
    val models: List<String>? = null,
    val selected: Set<String> = emptySet(),
)

class ChannelEditViewModel(private val container: AppContainer, private val channelId: Int?) : ViewModel() {
    private val api = container.api
    val isCreate: Boolean get() = channelId == null

    /** The form being edited. Snapshot state so text fields update synchronously. */
    var form by mutableStateOf(ChannelForm())
        private set

    /** What the form started from; the save sends only fields that differ from it. */
    private var baseline by mutableStateOf(ChannelForm())

    val isDirty: Boolean get() = form != baseline

    private val mutableState = MutableStateFlow(ChannelEditState(isLoading = channelId != null))
    val state: StateFlow<ChannelEditState> = mutableState.asStateFlow()

    private val mutableGroups = MutableStateFlow(GroupOptions())
    val groups: StateFlow<GroupOptions> = mutableGroups.asStateFlow()

    private val mutableUpstream = MutableStateFlow(UpstreamModels())
    val upstream: StateFlow<UpstreamModels> = mutableUpstream.asStateFlow()

    private val messageChannel = MessageChannel<UiMessage>(MessageChannel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    init {
        load()
        loadGroups()
    }

    fun load() {
        val id = channelId ?: return
        mutableState.update { it.copy(isLoading = true, loadError = null) }
        viewModelScope.launch {
            try {
                val channel = api.channel(id)
                val loaded = ChannelForm.of(channel)
                baseline = loaded
                form = loaded
                mutableState.update {
                    it.copy(isLoading = false, multiKeySize = if (channel.channelInfo.isMultiKey) channel.channelInfo.multiKeySize else 0)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, loadError = e) }
            }
        }
    }

    fun loadGroups() {
        val current = mutableGroups.value
        if (current.isLoading || (current.names.isNotEmpty() && !current.failed)) return
        mutableGroups.update { it.copy(isLoading = true, failed = false) }
        viewModelScope.launch {
            try {
                mutableGroups.value = GroupOptions(names = api.groups())
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableGroups.update { it.copy(isLoading = false, failed = true) }
            }
        }
    }

    fun update(transform: (ChannelForm) -> ChannelForm) {
        if (mutableState.value.isSaving) return
        form = transform(form)
    }

    fun toggleGroup(group: String) = update {
        it.copy(groups = if (group in it.groups) it.groups - group else it.groups + group)
    }

    fun removeModel(model: String) = update { current ->
        current.copy(models = current.modelList.filter { it != model }.joinToString(","))
    }

    /** True when the type was picked in this form (create, or a type change on edit). */
    val typeIsNew: Boolean get() = isCreate || form.type != baseline.type

    /**
     * [canEditSensitive]: type, key and base URL are only checked when the admin may change them,
     * so a read-only legacy value never blocks saving the other fields.
     */
    fun errors(canEditSensitive: Boolean): ChannelFormErrors {
        val current = form
        val required = R.string.common_required
        val priority = current.priority.trim()
        val weight = current.weight.trim()
        return ChannelFormErrors(
            name = required.takeIf { current.name.isBlank() },
            key = required.takeIf { isCreate && current.key.isBlank() },
            baseUrl = required.takeIf { canEditSensitive && current.type in ChannelTypes.baseUrlRequired && current.baseUrl.isBlank() },
            models = required.takeIf { current.modelList.isEmpty() },
            groups = when {
                current.groups.isEmpty() -> required
                current.groups.joinToString(",").length > MAX_GROUP_LENGTH -> R.string.channels_error_group_length
                else -> null
            },
            priority = R.string.channels_error_integer.takeIf { priority.isNotEmpty() && priority.toLongOrNull() == null },
            weight = R.string.channels_error_non_negative.takeIf { weight.isNotEmpty() && (weight.toLongOrNull() ?: -1) < 0 },
            remark = R.string.channels_error_remark_length.takeIf {
                current.remark.codePointCount(0, current.remark.length) > MAX_REMARK_LENGTH
            },
            modelMapping = R.string.channels_error_mapping.takeIf { !isStringMapping(current.modelMapping) },
        )
    }

    fun save(canEditSensitive: Boolean) {
        val current = mutableState.value
        if (current.isSaving || current.saved || current.isLoading || current.loadError != null) return
        if (!errors(canEditSensitive).isValid) {
            mutableState.update { it.copy(showErrors = true) }
            messageChannel.trySend(UiMessage.Res(R.string.channels_form_invalid))
            return
        }
        val submitted = form
        val id = channelId
        val patch = id?.let { changedFields(it, submitted, baseline) }
        if (patch == ChannelPatch(id = id ?: 0)) {
            // Only whitespace changed; nothing to send.
            mutableState.update { it.copy(saved = true) }
            return
        }
        mutableState.update { it.copy(isSaving = true, showErrors = true) }
        viewModelScope.launch {
            try {
                if (patch == null) {
                    api.createChannel(
                        ChannelDraft(
                            type = submitted.type,
                            name = submitted.name.trim(),
                            key = submitted.key,
                            baseUrl = submitted.baseUrl.trim(),
                            models = submitted.models,
                            group = submitted.groups.joinToString(","),
                            priority = submitted.priority.trim().toLongOrNull(),
                            weight = submitted.weight.trim().toLongOrNull(),
                            tag = submitted.tag.trim(),
                            remark = submitted.remark,
                            testModel = submitted.testModel.trim(),
                            modelMapping = submitted.modelMapping.trim(),
                            autoBan = submitted.autoBan,
                            batch = submitted.batch,
                        ),
                    )
                } else {
                    api.updateChannel(patch)
                }
                // A create has no id yet: the list reloads from the top.
                container.events.notifyChannelsChanged(id)
                mutableState.update { it.copy(isSaving = false, saved = true) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isSaving = false) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    fun fetchUpstreamModels() {
        val id = channelId ?: return
        if (mutableUpstream.value.isLoading) return
        mutableUpstream.value = UpstreamModels(isLoading = true)
        viewModelScope.launch {
            try {
                val models = api.fetchUpstreamModels(id).map { it.trim() }.filter { it.isNotEmpty() }.distinct()
                if (models.isEmpty()) {
                    mutableUpstream.value = UpstreamModels()
                    messageChannel.send(UiMessage.Res(R.string.channels_fetch_models_empty))
                } else {
                    val current = form.modelList.toSet()
                    mutableUpstream.value = UpstreamModels(models = models, selected = models.filterTo(LinkedHashSet()) { it in current })
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableUpstream.value = UpstreamModels()
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    fun toggleUpstreamModel(model: String) = mutableUpstream.update {
        it.copy(selected = if (model in it.selected) it.selected - model else it.selected + model)
    }

    /** Checks or unchecks every model in [models] (the picker's filtered view). */
    fun setUpstreamSelection(models: List<String>, selected: Boolean) = mutableUpstream.update {
        it.copy(selected = if (selected) it.selected + models else it.selected - models.toSet())
    }

    fun dismissUpstreamModels() {
        mutableUpstream.value = UpstreamModels()
    }

    /**
     * Merges the picker into the models field: current models the upstream does not list are kept, upstream
     * models follow their checkbox, and newly checked ones are appended.
     */
    fun applyUpstreamModels() {
        val picker = mutableUpstream.value
        val fetched = picker.models ?: return
        val fetchedSet = fetched.toSet()
        update { current ->
            val existing = current.modelList
            val kept = existing.filter { it !in fetchedSet || it in picker.selected }
            val added = fetched.filter { it in picker.selected && it !in existing }
            current.copy(models = (kept + added).joinToString(","))
        }
        mutableUpstream.value = UpstreamModels()
    }
}

/** `model_mapping` must be blank or a JSON object whose values are all strings (`{"from": "to"}`). */
private fun isStringMapping(value: String): Boolean {
    val text = value.trim()
    if (text.isEmpty()) return true
    val element = runCatching { Json.parseToJsonElement(text) }.getOrNull() as? JsonObject ?: return false
    return element.values.all { it is JsonPrimitive && it.isString }
}

/**
 * `PUT /api/channel/` body with only the fields that differ from [baseline]. The server rejects
 * unchanged sensitive fields for admins without `sensitive_write`, so nothing unchanged is sent.
 */
private fun changedFields(id: Int, form: ChannelForm, baseline: ChannelForm): ChannelPatch {
    val typeChanged = form.type != baseline.type
    val baseUrl = form.baseUrl.trim()
    val priority = form.priority.trim()
    val weight = form.weight.trim()
    return ChannelPatch(
        id = id,
        name = form.name.trim().takeIf { it != baseline.name.trim() },
        models = form.modelList.joinToString(",").takeIf { it != baseline.modelList.joinToString(",") },
        group = form.groups.joinToString(",").takeIf { it != baseline.groups.joinToString(",") },
        // A cleared number is sent as 0: the server keeps omitted fields.
        priority = (priority.toLongOrNull() ?: 0L).takeIf { priority != baseline.priority.trim() },
        weight = (weight.toLongOrNull() ?: 0L).takeIf { weight != baseline.weight.trim() },
        tag = form.tag.trim().takeIf { it != baseline.tag.trim() },
        remark = form.remark.takeIf { it != baseline.remark },
        testModel = form.testModel.trim().takeIf { it != baseline.testModel.trim() },
        modelMapping = form.modelMapping.trim().takeIf { it != baseline.modelMapping.trim() },
        autoBan = form.autoBan.takeIf { it != baseline.autoBan },
        // An unchanged type is not a sensitive change, and a Codex key is only validated with the type in the body.
        type = form.type.takeIf { typeChanged || (form.type == TYPE_CODEX && form.key.isNotBlank()) },
        key = form.key.trim().takeIf { it.isNotEmpty() },
        // The server validates type 60 against the request body alone, so a switch to it carries the URL.
        baseUrl = baseUrl.takeIf { it != baseline.baseUrl.trim() || (typeChanged && form.type == TYPE_NEW_API) },
    )
}
