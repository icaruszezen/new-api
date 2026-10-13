package com.newapi.admin.ui

import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ReceiptLong
import androidx.compose.material.icons.outlined.Group
import androidx.compose.material.icons.outlined.Hub
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.collectAsStateWithLifecycle
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
import com.newapi.admin.ui.navigation.SettingsRoute
import com.newapi.admin.ui.navigation.UserCreateRoute
import com.newapi.admin.ui.navigation.UserDetailRoute
import com.newapi.admin.ui.navigation.UsersRoute
import com.newapi.admin.ui.settings.SettingsScreen
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.users.UserCreateScreen
import com.newapi.admin.ui.users.UserDetailScreen
import com.newapi.admin.ui.users.UsersScreen

private enum class TopLevelTab(val route: Any, val label: Int, val icon: ImageVector) {
    Logs(LogsRoute, R.string.nav_logs, Icons.AutoMirrored.Outlined.ReceiptLong),
    Users(UsersRoute, R.string.nav_users, Icons.Outlined.Group),
    Channels(ChannelsRoute, R.string.nav_channels, Icons.Outlined.Hub),
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
}

@Composable
private fun MainScaffold() {
    val navController = rememberNavController()
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
                    modifier = Modifier.drawBehind {
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
                            icon = { Icon(tab.icon, contentDescription = null) },
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
            composable<SettingsRoute> { SettingsScreen() }
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
