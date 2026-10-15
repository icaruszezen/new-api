package com.newapi.admin.ui.tickets

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.newapi.admin.R
import com.newapi.admin.core.model.TicketStatus
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.Tone

/** The statuses a reply can set besides "automatic". Like the web, a reply never moves a ticket back to awaiting support. */
internal val REPLY_STATUSES = listOf(TicketStatus.PROCESSING, TicketStatus.RESOLVED, TicketStatus.CLOSED)

@Composable
internal fun ticketStatusLabel(status: Int): String = when (status) {
    TicketStatus.PENDING -> stringResource(R.string.tickets_status_pending)
    TicketStatus.PROCESSING -> stringResource(R.string.tickets_status_processing)
    TicketStatus.RESOLVED -> stringResource(R.string.tickets_status_resolved)
    TicketStatus.CLOSED -> stringResource(R.string.tickets_status_closed)
    else -> stringResource(R.string.tickets_status_unknown, status)
}

@Composable
internal fun TicketStatusPill(status: Int, modifier: Modifier = Modifier) {
    val tone = when (status) {
        TicketStatus.PENDING -> Tone.Warning
        TicketStatus.PROCESSING -> Tone.Info
        TicketStatus.RESOLVED -> Tone.Success
        else -> Tone.Neutral
    }
    StatusPill(ticketStatusLabel(status), tone, modifier)
}
