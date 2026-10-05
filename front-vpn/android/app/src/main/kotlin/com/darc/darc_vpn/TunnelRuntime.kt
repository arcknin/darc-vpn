package com.darc.darc_vpn

import android.content.ComponentName
import android.content.Context
import android.content.ContextWrapper
import android.content.Intent
import android.os.Build
import android.os.Handler
import android.os.Looper
import com.wireguard.android.backend.GoBackend
import com.wireguard.android.backend.Tunnel
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicLong

/** Process-local control; never persist/log configuration or serialize secrets. */
object TunnelRuntime {
    val executor = Executors.newSingleThreadExecutor()
    val generation = AtomicLong()
    private val main = Handler(Looper.getMainLooper())
    private var instance: GoBackend? = null
    @Volatile var observer: ((String) -> Unit)? = null
    val tunnel = object : Tunnel {
        override fun getName() = "darcvpn"
        override fun onStateChange(newState: Tunnel.State) {
            emit(if (newState == Tunnel.State.UP) "active" else "disconnected")
        }
    }
    fun emit(state: String) { main.post { observer?.invoke(state) } }
    fun backend(context: Context): GoBackend {
        val current = instance
        if (current != null) return current
        val serviceContext = object : ContextWrapper(context.applicationContext) {
            override fun startService(service: Intent): ComponentName? {
                if (service.component?.className == GoBackend.VpnService::class.java.name) {
                    val foreground = Intent(this, DarcVpnService::class.java)
                    return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O)
                        baseContext.startForegroundService(foreground) else baseContext.startService(foreground)
                }
                return baseContext.startService(service)
            }
        }
        return GoBackend(serviceContext).also { instance = it }
    }
    fun currentState(context: Context): String {
        if (instance == null) return "disconnected"
        return if (backend(context).getState(tunnel) == Tunnel.State.UP) "active" else "disconnected"
    }
}
