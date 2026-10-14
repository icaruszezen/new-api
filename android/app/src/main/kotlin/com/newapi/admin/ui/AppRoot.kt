package com.newapi.admin.ui

import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ReceiptLong
import androidx.compose.material.icons.outlined.Group
import androidx.compose.material.icons.outlined.Hub
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Redeem
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LifecycleEventEffect
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.repeatOnLifecycle
import androidx.lifecycle.viewmodel.compose.LocalViewModelStoreOwner
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavBackStackEntry
import androidx.navigation.NavController
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.auth.AuthState
import com.newapi.admin.core.format.Formats
import com.newapi.admin.ui.channels.ChannelDetailScreen
import com.newapi.admin.ui.channels.ChannelEditScreen
import com.newapi.admin.ui.channels.ChannelsScreen
import com.newapi.admin.ui.login.LoginScreen
import com.newapi.admin.ui.logs.LogsScreen
import com.newapi.admin.ui.navigation.ChannelDetailRoute
import com.newapi.admin.ui.navigation.ChannelEditRoute
import com.newapi.admin.ui.navigation.ChannelsRoute
import com.newapi.admin.ui.navigation.FilteredLogsRoute
import com.newapi.admin.ui.navigation.LogsRoute
import com.newapi.admin.ui.navigation.RedemptionDetailRoute
import com.newapi.admin.ui.navigation.RedemptionEditRoute
import com.newapi.admin.ui.navigation.RedemptionsRoute
import com.newapi.admin.ui.navigation.SettingsRoute
import com.newapi.admin.ui.navigation.TicketDetailRoute
import com.newapi.admin.ui.navigation.TicketsRoute
import com.newapi.admin.ui.navigation.UserCreateRoute
import com.newapi.admin.ui.navigation.UserDetailRoute
import com.newapi.admin.ui.navigation.UsersRoute
import com.newapi.admin.ui.redemptions.RedemptionDetailScreen
import com.newapi.admin.ui.redemptions.RedemptionEditScreen
import com.newapi.admin.ui.redemptions.RedemptionsScreen
import com.newapi.admin.ui.settings.SettingsScreen
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.tickets.TicketDetailScreen
import com.newapi.admin.ui.tickets.TicketsScreen
import com.newapi.admin.ui.update.UpdatePrompt
import com.newapi.admin.ui.users.UserCreateScreen
import com.newapi.admin.ui.users.UserDetailScreen
import com.newapi.admin.ui.users.UsersScreen
import kotlinx.coroutines.delay

private enum class TopLevelTab(val route: Any, val label: Int, val icon: ImageVector) {
    Logs(LogsRoute, R.string.nav_logs, Icons.AutoMirrored.Outlined.ReceiptLong),
    Users(UsersRoute, R.string.nav_users, Icons.Outlined.Group),
    Channels(ChannelsRoute, R.string.nav_channels, Icons.Outlined.Hub),
    Redemptions(RedemptionsRoute, R.string.nav_redemptions, Icons.Outlined.Redeem),
    Me(SettingsRoute, R.string.nav_me, Icons.Outlined.Person),
}

/** Login gate, then the tabbed admin UI. */
@Composable
fun AppRoot() {
    val container = LocalAppContainer.current
    val authState by container.auth.state.collectAsStateWithLifecycle()
    val sessionViewModels = viewModel { SessionViewModels() }
    val account = (authState as? AuthState.SignedIn)?.account
    sessionViewModels.bind(account?.let { it.baseUrl to it.user.id })
    // Runs after the previous session's screens were disposed in this same frame.
    SideEffect { sessionViewModels.clearRetired() }
    when (val state = authState) {
        is AuthState.SignedOut -> LoginScreen(signedOutReason = state.reason)
        // A different account or server starts from a fresh back stack and fresh ViewModels.
        is AuthState.SignedIn -> key(state.account.baseUrl, state.account.user.id) {
            CompositionLocalProvider(LocalViewModelStoreOwner provides sessionViewModels) { MainScaffold() }
        }
    }
    // Updates are app-wide, so the dialog also shows over the login screen.
    LifecycleEventEffect(Lifecycle.Event.ON_START) { container.updates.onForeground() }
    UpdatePrompt()
}

/** How often the ticket counts behind the Me tab badge reload while the app is on screen. */
private const val TICKET_STATS_INTERVAL_MILLIS = 60_000L

@Composable
private fun MainScaffold() {
    val container = LocalAppContainer.current
    val navController = rememberNavController()
    val ticketStats by container.ticketStats.collectAsStateWithLifecycle()
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(lifecycle) {
        // There are no push notifications: new tickets show up through this reload, also right after returning.
        lifecycle.repeatOnLifecycle(Lifecycle.State.STARTED) {
            while (true) {
                container.refreshTicketStats()
                delay(TICKET_STATS_INTERVAL_MILLIS)
            }
        }
    }
    val backStackEntry by navController.currentBackStackEntryAsState()
    val destination = backStackEntry?.destination
    val selectedTab = TopLevelTab.entries.firstOrNull { tab ->
        destination?.hierarchy?.any { it.hasRoute(tab.route::class) } == true
    }
    Scaffold(
        // Each screen draws its own top bar and handles the remaining insets.
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        bottomBar = {
            if (destination == null || selectedTab != null) {
                val colors = AppTheme.colors
                NavigationBar(
                    modifier = Modifier.drawWithContent {
                        drawContent()
                        drawLine(colors.border, Offset(0f, 0.5.dp.toPx()), Offset(size.width, 0.5.dp.toPx()), strokeWidth = 1.dp.toPx())
                    },
                    containerColor = colors.background,
                    tonalElevation = 0.dp,
                ) {
                    TopLevelTab.entries.forEach { tab ->
                        NavigationBarItem(
                            selected = tab == selectedTab,
                            onClick = {
                                navController.navigate(tab.route) {
                                    popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            icon = {
                                val unread = ticketStats?.unread ?: 0
                                if (tab == TopLevelTab.Me && unread > 0) {
                                    val description = pluralStringResource(
                                        R.plurals.tickets_badge_description,
                                        unread.coerceAtMost(Int.MAX_VALUE.toLong()).toInt(),
                                        Formats.count(unread),
                                    )
                                    BadgedBox(
                                        badge = {
                                            Badge(
                                                containerColor = colors.primary,
                                                contentColor = colors.primaryForeground,
                                                modifier = Modifier.semantics { contentDescription = description },
                                            ) {
                                                Text(if (unread > 99) "99+" else unread.toString())
                                            }
                                        },
                                    ) {
                                        Icon(tab.icon, contentDescription = null)
                                    }
                                } else {
                                    Icon(tab.icon, contentDescription = null)
                                }
                            },
                            label = { Text(stringResource(tab.label), style = MaterialTheme.typography.labelMedium) },
                            colors = NavigationBarItemDefaults.colors(
                                selectedIconColor = colors.foreground,
                                selectedTextColor = colors.foreground,
                                indicatorColor = colors.accent,
                                unselectedIconColor = colors.mutedForeground,
                                unselectedTextColor = colors.mutedForeground,
                            ),
                        )
                    }
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = LogsRoute,
            modifier = Modifier.padding(padding).consumeWindowInsets(padding),
        ) {
            composable<LogsRoute> { entry ->
                LogsScreen(
                    onOpenUser = { navController.navigateFrom(entry, UserDetailRoute(it)) },
                    onOpenChannel = { navController.navigateFrom(entry, ChannelDetailRoute(it)) },
                )
            }
            composable<FilteredLogsRoute> { entry ->
                val route = entry.toRoute<FilteredLogsRoute>()
                LogsScreen(
                    initialUsername = route.username.ifEmpty { null },
                    initialChannelId = route.channelId.takeIf { it > 0 },
                    onBack = { navController.backFrom(entry) },
                    onOpenUser = { navController.navigateFrom(entry, UserDetailRoute(it)) },
                    onOpenChannel = { navController.navigateFrom(entry, ChannelDetailRoute(it)) },
                )
            }
            composable<UsersRoute> { entry ->
                UsersScreen(
                    onOpenUser = { navController.navigateFrom(entry, UserDetailRoute(it)) },
                    onCreateUser = { navController.navigateFrom(entry, UserCreateRoute) },
                )
            }
            composable<UserDetailRoute> { entry ->
                UserDetailScreen(
                    userId = entry.toRoute<UserDetailRoute>().id,
                    onBack = { navController.backFrom(entry) },
                    onOpenLogs = { navController.navigateFrom(entry, FilteredLogsRoute(username = it)) },
                )
            }
            composable<UserCreateRoute> { entry ->
                UserCreateScreen(
                    onBack = { navController.backFrom(entry) },
                    onCreated = { navController.backFrom(entry) },
                )
            }
            composable<ChannelsRoute> { entry ->
                ChannelsScreen(
                    onOpenChannel = { navController.navigateFrom(entry, ChannelDetailRoute(it)) },
                    onCreateChannel = { navController.navigateFrom(entry, ChannelEditRoute()) },
                )
            }
            composable<ChannelDetailRoute> { entry ->
                ChannelDetailScreen(
                    channelId = entry.toRoute<ChannelDetailRoute>().id,
                    onBack = { navController.backFrom(entry) },
                    onEdit = { navController.navigateFrom(entry, ChannelEditRoute(it)) },
                    onOpenLogs = { navController.navigateFrom(entry, FilteredLogsRoute(channelId = it)) },
                )
            }
            composable<ChannelEditRoute> { entry ->
                ChannelEditScreen(
                    channelId = entry.toRoute<ChannelEditRoute>().id.takeIf { it > 0 },
                    onBack = { navController.backFrom(entry) },
                    onSaved = { navController.backFrom(entry) },
                )
            }
            composable<RedemptionsRoute> { entry ->
                RedemptionsScreen(
                    onOpenCode = { navController.navigateFrom(entry, RedemptionDetailRoute(it)) },
                    onCreateCodes = { navController.navigateFrom(entry, RedemptionEditRoute()) },
                )
            }
            composable<RedemptionDetailRoute> { entry ->
                RedemptionDetailScreen(
                    codeId = entry.toRoute<RedemptionDetailRoute>().id,
                    onBack = { navController.backFrom(entry) },
                    onEdit = { navController.navigateFrom(entry, RedemptionEditRoute(it)) },
                    onOpenUser = { navController.navigateFrom(entry, UserDetailRoute(it)) },
                )
            }
            composable<RedemptionEditRoute> { entry ->
                RedemptionEditScreen(
                    codeId = entry.toRoute<RedemptionEditRoute>().id.takeIf { it > 0 },
                    onBack = { navController.backFrom(entry) },
                    onDone = { navController.backFrom(entry) },
                )
            }
            composable<SettingsRoute> { entry ->
                SettingsScreen(onOpenTickets = { navController.navigateFrom(entry, TicketsRoute) })
            }
            composable<TicketsRoute> { entry ->
                TicketsScreen(
                    onBack = { navController.backFrom(entry) },
                    onOpenTicket = { navController.navigateFrom(entry, TicketDetailRoute(it)) },
                )
            }
            composable<TicketDetailRoute> { entry ->
                TicketDetailScreen(
                    ticketId = entry.toRoute<TicketDetailRoute>().id,
                    onBack = { navController.backFrom(entry) },
                    onOpenUser = { navController.navigateFrom(entry, UserDetailRoute(it)) },
                )
            }
        }
    }
}

// Taps on a screen that is already leaving are ignored, so a double tap cannot navigate twice or pop past the start.

private fun NavController.navigateFrom(entry: NavBackStackEntry, route: Any) {
    if (entry.lifecycle.currentState == Lifecycle.State.RESUMED) navigate(route)
}

private fun NavController.backFrom(entry: NavBackStackEntry) {
    if (entry.lifecycle.currentState == Lifecycle.State.RESUMED) popBackStack()
}
