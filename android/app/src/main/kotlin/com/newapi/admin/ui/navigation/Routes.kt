package com.newapi.admin.ui.navigation

import kotlinx.serialization.Serializable

// Type-safe navigation routes. Optional filters use empty/zero sentinels instead of nullable
// primitives so the generated NavTypes stay simple.

@Serializable data object LogsRoute

/** Usage logs pre-filtered from a user or channel detail screen. */
@Serializable data class FilteredLogsRoute(val username: String = "", val channelId: Int = 0)

@Serializable data object UsersRoute

@Serializable data class UserDetailRoute(val id: Int)

@Serializable data object UserCreateRoute

@Serializable data object ChannelsRoute

@Serializable data class ChannelDetailRoute(val id: Int)

/** Edits channel [id], or creates a channel when [id] is 0. */
@Serializable data class ChannelEditRoute(val id: Int = 0)

@Serializable data object RedemptionsRoute

@Serializable data class RedemptionDetailRoute(val id: Int)

/** Edits redemption code [id], or creates codes when [id] is 0. */
@Serializable data class RedemptionEditRoute(val id: Int = 0)

@Serializable data object SettingsRoute
