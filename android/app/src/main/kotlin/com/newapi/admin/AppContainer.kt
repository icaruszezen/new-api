package com.newapi.admin

import androidx.compose.runtime.staticCompositionLocalOf
import com.newapi.admin.core.api.AdminApi
import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.auth.AuthState
import com.newapi.admin.core.auth.SessionStore
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.SelfUser
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.model.Ticket
import com.newapi.admin.core.model.TicketStats
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.NewApiClient
import com.newapi.admin.core.update.AppUpdater
import com.newapi.admin.data.AppPreferences
import com.newapi.admin.update.UpdateController
import java.io.File
import java.util.Locale
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.onSubscription
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/** App-wide singletons. Created once by the Application; screens reach it through [LocalAppContainer]. */
class AppContainer(
    sessionStore: SessionStore,
    val preferences: AppPreferences,
    userAgent: String,
    /** versionName and versionCode, shown on the settings screen. */
    val appVersion: String,
    val appVersionCode: Int,
    /** Null for builds that cannot update themselves (see [UpdateController]). */
    updater: AppUpdater?,
    /** Private cache directory for downloaded updates. */
    updateDir: File,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val client = NewApiClient(
        store = sessionStore,
        userAgent = userAgent,
        acceptLanguage = { Locale.getDefault().toLanguageTag() },
    )

    val auth: AuthManager get() = client.auth
    val api: AdminApi get() = client.admin

    /** Signed-in admin, or null. */
    val currentUser: StateFlow<SelfUser?> = auth.state
        .map { (it as? AuthState.SignedIn)?.account?.user }
        .stateIn(scope, SharingStarted.Eagerly, (auth.state.value as? AuthState.SignedIn)?.account?.user)

    private val statusFlow = MutableStateFlow<SystemStatus?>(null)

    /** `/api/status` of the signed-in server (display currency, system name, version). */
    val status: StateFlow<SystemStatus?> = statusFlow.asStateFlow()

    val quotaFormatter: StateFlow<QuotaFormatter> = statusFlow
        .map { status -> status?.let(::QuotaFormatter) ?: QuotaFormatter.Default }
        .stateIn(scope, SharingStarted.Eagerly, QuotaFormatter.Default)

    private val ticketStatsFlow = MutableStateFlow<TicketStats?>(null)

    /** Ticket counts of the signed-in server; null until loaded, or when the server has no ticket system. */
    val ticketStats: StateFlow<TicketStats?> = ticketStatsFlow.asStateFlow()

    private val ticketsSupportedFlow = MutableStateFlow(true)

    /** False once the server answered the ticket stats with 404: a new-api without the ticket system. */
    val ticketsSupported: StateFlow<Boolean> = ticketsSupportedFlow.asStateFlow()

    private val ticketStatsRequests = MutableSharedFlow<Unit>(extraBufferCapacity = 1, onBufferOverflow = BufferOverflow.DROP_OLDEST)

    val events = AppEvents()

    val updates = UpdateController(updater, preferences, appVersionCode, updateDir, scope)

    init {
        updates.onAppStart()
        scope.launch {
            auth.state
                .map { (it as? AuthState.SignedIn)?.account?.baseUrl }
                .distinctUntilChanged()
                .collect { baseUrl ->
                    statusFlow.value = null
                    if (baseUrl != null) refreshStatus()
                }
        }
        // A new account drops the old counts and cancels a stats request still running for it.
        scope.launch {
            auth.state
                .map { (it as? AuthState.SignedIn)?.account?.let { account -> account.baseUrl to account.user.id } }
                .distinctUntilChanged()
                .collectLatest { account ->
                    ticketStatsFlow.value = null
                    ticketsSupportedFlow.value = true
                    if (account == null) return@collectLatest
                    // A newer request replaces one still running, so a slow old answer cannot overwrite a fresh one.
                    ticketStatsRequests.onSubscription { emit(Unit) }.collectLatest { loadTicketStats() }
                }
        }
    }

    fun refreshStatus() {
        scope.launch {
            runCatching { api.status() }.onSuccess { statusFlow.value = it }
        }
    }

    /** Reloads [ticketStats] (the unread badge and per-status counts) in the background. */
    fun refreshTicketStats() {
        ticketStatsRequests.tryEmit(Unit)
    }

    private suspend fun loadTicketStats() {
        try {
            ticketStatsFlow.value = api.ticketStats()
            ticketsSupportedFlow.value = true
        } catch (e: CancellationException) {
            throw e
        } catch (e: ApiError.Http) {
            if (e.status == 404) {
                ticketStatsFlow.value = null
                ticketsSupportedFlow.value = false
            }
        } catch (e: Throwable) {
            // Keep the last counts; the next refresh tries again.
        }
    }
}

/**
 * Lets list screens catch up after a detail/edit screen changed something. The payload is the id of
 * the changed (or deleted) row, or null when the list should reload from the top (a row was created).
 * Tickets carry the changed row itself, since the only way to fetch one also marks it read.
 */
class AppEvents {
    private val users = MutableSharedFlow<Int?>(extraBufferCapacity = 8)
    private val channels = MutableSharedFlow<Int?>(extraBufferCapacity = 8)
    private val redemptions = MutableSharedFlow<Int?>(extraBufferCapacity = 8)
    private val tickets = MutableSharedFlow<Ticket?>(extraBufferCapacity = 8)

    val usersChanged: SharedFlow<Int?> = users.asSharedFlow()
    val channelsChanged: SharedFlow<Int?> = channels.asSharedFlow()
    val redemptionsChanged: SharedFlow<Int?> = redemptions.asSharedFlow()
    val ticketsChanged: SharedFlow<Ticket?> = tickets.asSharedFlow()

    fun notifyUsersChanged(id: Int? = null) {
        users.tryEmit(id)
    }

    fun notifyChannelsChanged(id: Int? = null) {
        channels.tryEmit(id)
    }

    fun notifyRedemptionsChanged(id: Int? = null) {
        redemptions.tryEmit(id)
    }

    /** [ticket] as it is now after it was opened (read), answered or moved to another status; null reloads the list. */
    fun notifyTicketsChanged(ticket: Ticket? = null) {
        tickets.tryEmit(ticket)
    }
}

val LocalAppContainer = staticCompositionLocalOf<AppContainer> { error("AppContainer not provided") }
