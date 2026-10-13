package com.newapi.admin

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.CompositionLocalProvider
import com.newapi.admin.ui.AppRoot
import com.newapi.admin.ui.theme.AdminTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val container = (application as NewApiAdminApp).container
        setContent {
            CompositionLocalProvider(LocalAppContainer provides container) {
                AdminTheme { AppRoot() }
            }
        }
    }
}
