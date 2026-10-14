package com.newapi.admin.ui.redemptions

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.users.runeCount
import java.math.BigDecimal
import java.math.RoundingMode
import java.time.Instant
import java.time.ZoneId
import kotlin.math.abs
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** Expiry presets. Like the web's quick buttons they count from the moment they are picked; a month is a calendar month. */
enum class ExpiryChoice { NEVER, DAY, WEEK, MONTH, CUSTOM }

/** The form as typed. Numbers stay text so partial input survives. */
data class RedemptionForm(
    val name: String = "",
    /** In the display currency (raw units when the display type is TOKENS); the web's default is 10. */
    val amount: String = "10",
    val count: String = "1",
    val expiry: ExpiryChoice = ExpiryChoice.NEVER,
    /** Unix seconds; 0 never expires. */
    val expiresAt: Long = 0,
)

data class RedemptionFormErrors(
    val name: UiMessage.Res? = null,
    val amount: UiMessage.Res? = null,
    val count: UiMessage.Res? = null,
    val expiry: UiMessage.Res? = null,
) {
    val isValid: Boolean get() = name == null && amount == null && count == null && expiry == null
}

data class RedemptionEditState(
    /** Edit only: loading the code. */
    val isLoading: Boolean = false,
    val loadError: Throwable? = null,
    val isSaving: Boolean = false,
    /** Field errors are shown after the first save attempt. */
    val showErrors: Boolean = false,
    /** Create: the codes the server generated, shown until the admin leaves. */
    val createdKeys: List<String>? = null,
    /** Edit: saved; the screen leaves. */
    val saved: Boolean = false,
)

/** Creates codes when [codeId] is null, otherwise edits the name, amount and expiry of that code. */
class RedemptionEditViewModel(private val container: AppContainer, private val codeId: Int?) : ViewModel() {
    private val api = container.api
    val isCreate: Boolean get() = codeId == null

    /** The form being edited. Snapshot state so text fields update synchronously. */
    var form by mutableStateOf(RedemptionForm())
        private set

    /** What the form started from. */
    private var baseline by mutableStateOf(RedemptionForm())

    val isDirty: Boolean get() = form != baseline

    /** Edit: the stored quota, sent back as it is while the amount is untouched so a rounded display value cannot change it. */
    private var storedQuota = 0L

    private val mutableState = MutableStateFlow(RedemptionEditState(isLoading = codeId != null))
    val state: StateFlow<RedemptionEditState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    init {
        load()
    }

    fun load() {
        val id = codeId ?: return
        mutableState.update { it.copy(isLoading = true, loadError = null) }
        viewModelScope.launch {
            try {
                val code = api.redemption(id)
                // The amount is edited in the display currency, so the server's settings are needed first.
                val formatter = QuotaFormatter(container.status.filterNotNull().first())
                // Like the web's quotaUnitsToEditableAmount: 2 decimals from 1 up, 4 below.
                val amount = formatter.toAmount(code.quota)
                val amountText = if (formatter.isTokens) {
                    code.quota.toString()
                } else {
                    BigDecimal.valueOf(amount).setScale(if (abs(amount) >= 1) 2 else 4, RoundingMode.HALF_UP).stripTrailingZeros().toPlainString()
                }
                val loaded = RedemptionForm(
                    name = code.name,
                    amount = amountText,
                    expiry = if (code.expiredTime == 0L) ExpiryChoice.NEVER else ExpiryChoice.CUSTOM,
                    expiresAt = code.expiredTime,
                )
                storedQuota = code.quota
                baseline = loaded
                form = loaded
                mutableState.update { it.copy(isLoading = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, loadError = e) }
            }
        }
    }

    fun update(transform: (RedemptionForm) -> RedemptionForm) {
        val current = mutableState.value
        if (current.isSaving || current.createdKeys != null) return
        form = transform(form)
    }

    /** A preset; [ExpiryChoice.CUSTOM] goes through [setCustomExpiry]. */
    fun chooseExpiry(choice: ExpiryChoice) = update {
        val now = Instant.now().atZone(ZoneId.systemDefault())
        val expiresAt = when (choice) {
            ExpiryChoice.NEVER, ExpiryChoice.CUSTOM -> 0L
            ExpiryChoice.DAY -> now.plusDays(1).toEpochSecond()
            ExpiryChoice.WEEK -> now.plusWeeks(1).toEpochSecond()
            ExpiryChoice.MONTH -> now.plusMonths(1).toEpochSecond()
        }
        it.copy(expiry = choice, expiresAt = expiresAt)
    }

    fun setCustomExpiry(epochSeconds: Long) = update { it.copy(expiry = ExpiryChoice.CUSTOM, expiresAt = epochSeconds) }

    /** The quota a save sends: the stored one while an edit leaves the amount untouched, else the typed amount; null if invalid. */
    fun savedQuota(formatter: QuotaFormatter): Long? {
        if (!isCreate && form.amount == baseline.amount) return storedQuota
        return (CodeAmount.parse(form.amount, formatter) as? CodeAmount.Valid)?.quota
    }

    fun errors(formatter: QuotaFormatter, nowSeconds: Long): RedemptionFormErrors {
        val current = form
        // An untouched amount is sent back as stored, so its rounded display value is not checked.
        val amountUnchanged = !isCreate && current.amount == baseline.amount
        val amount = CodeAmount.parse(current.amount, formatter)
        val name = current.name.trim()
        // A blank name on create becomes the formatted amount, as on the web.
        val sentName = if (isCreate && name.isEmpty() && amount is CodeAmount.Valid) formatter.format(amount.quota) else name
        val count = current.count.trim().toIntOrNull()
        return RedemptionFormErrors(
            name = when {
                sentName.isEmpty() -> if (isCreate) null else UiMessage.Res(R.string.common_required)
                sentName.runeCount() > MAX_CODE_NAME_LENGTH -> UiMessage.Res(R.string.redemptions_error_too_long, listOf(MAX_CODE_NAME_LENGTH))
                else -> null
            },
            amount = when {
                amountUnchanged -> null
                amount is CodeAmount.Empty -> UiMessage.Res(R.string.common_required)
                amount is CodeAmount.Invalid -> amount.message
                else -> null
            },
            count = UiMessage.Res(R.string.redemptions_error_count).takeIf { isCreate && (count == null || count !in CODE_COUNT_RANGE) },
            expiry = UiMessage.Res(R.string.redemptions_error_expiry_past).takeIf { current.expiresAt != 0L && current.expiresAt <= nowSeconds },
        )
    }

    fun save(formatter: QuotaFormatter) {
        val current = mutableState.value
        if (current.isSaving || current.saved || current.createdKeys != null || current.isLoading || current.loadError != null) return
        if (!errors(formatter, System.currentTimeMillis() / 1000).isValid) {
            mutableState.update { it.copy(showErrors = true) }
            messageChannel.trySend(UiMessage.Res(R.string.redemptions_form_invalid))
            return
        }
        val submitted = form
        val id = codeId
        if (id != null && submitted == baseline) {
            mutableState.update { it.copy(saved = true) }
            return
        }
        val quota = savedQuota(formatter) ?: return
        val name = submitted.name.trim().ifEmpty { formatter.format(quota) }
        mutableState.update { it.copy(isSaving = true, showErrors = true) }
        viewModelScope.launch {
            try {
                if (id == null) {
                    val keys = api.createRedemptions(name, quota, submitted.count.trim().toInt(), submitted.expiresAt)
                    container.events.notifyRedemptionsChanged()
                    mutableState.update { it.copy(isSaving = false, createdKeys = keys) }
                } else {
                    api.updateRedemption(id, name, quota, submitted.expiresAt)
                    container.events.notifyRedemptionsChanged(id)
                    mutableState.update { it.copy(isSaving = false, saved = true) }
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // A create that fails part-way keeps the codes made before the failure.
                if (id == null) container.events.notifyRedemptionsChanged()
                mutableState.update { it.copy(isSaving = false) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }
}
