package com.darc.darc_vpn

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import android.os.Handler
import android.os.Looper
import com.wireguard.android.backend.Tunnel
import com.wireguard.config.Config
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val main = Handler(Looper.getMainLooper())
    private var permissionResult: MethodChannel.Result? = null
    private var sink: EventChannel.EventSink? = null
    private var listening = false
    private val poll = object : Runnable {
        override fun run() {
            if (!listening) return
            TunnelRuntime.executor.execute {
                try { TunnelRuntime.emit(TunnelRuntime.currentState(applicationContext)) }
                catch (_: Exception) { TunnelRuntime.emit("failed") }
            }
            main.postDelayed(this, 1500)
        }
    }
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, "darc.vpn/tunnel-state").setStreamHandler(object : EventChannel.StreamHandler {
            override fun onListen(arguments: Any?, events: EventChannel.EventSink) {
                sink = events; listening = true
                TunnelRuntime.observer = { state -> sink?.success(state) }
                main.post(poll)
            }
            override fun onCancel(arguments: Any?) { listening = false; main.removeCallbacks(poll); sink = null; TunnelRuntime.observer = null }
        })
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "darc.vpn/tunnel").setMethodCallHandler { call, result ->
            when (call.method) {
                "preparePermission" -> {
                    if (permissionResult != null) result.error("busy", "Uma solicitação de VPN já está em andamento.", null)
                    else {
                        val intent = VpnService.prepare(this)
                        if (intent == null) result.success(true)
                        else { permissionResult = result; startActivityForResult(intent, 4102) }
                    }
                }
                "start" -> {
                    val configuration = call.argument<String>("configuration")
                    if (configuration.isNullOrBlank() || configuration.length > 65536) { result.error("invalid_configuration", "Configuração VPN inválida.", null) }
                    else if (VpnService.prepare(this) != null) { result.error("permission_denied", "Permissão de VPN não concedida.", null) }
                    else {
                        val operation = TunnelRuntime.generation.incrementAndGet()
                        TunnelRuntime.emit("connecting")
                        TunnelRuntime.executor.execute {
                            try {
                                if (operation != TunnelRuntime.generation.get()) { main.post { result.error("cancelled", "Conexão cancelada.", null) }; return@execute }
                                val config = configuration.byteInputStream().use { Config.parse(it) }
                                val backend = TunnelRuntime.backend(applicationContext)
                                backend.setState(TunnelRuntime.tunnel, Tunnel.State.UP, config)
                                if (operation != TunnelRuntime.generation.get()) backend.setState(TunnelRuntime.tunnel, Tunnel.State.DOWN, null)
                                TunnelRuntime.emit(TunnelRuntime.currentState(applicationContext))
                                main.post { result.success(null) }
                            } catch (_: Exception) {
                                TunnelRuntime.emit("failed")
                                main.post { result.error("tunnel_failed", "Não foi possível iniciar a VPN.", null) }
                            }
                        }
                    }
                }
                "stop" -> {
                    TunnelRuntime.generation.incrementAndGet()
                    TunnelRuntime.emit("disconnecting")
                    TunnelRuntime.executor.execute {
                        try {
                            TunnelRuntime.backend(applicationContext).setState(TunnelRuntime.tunnel, Tunnel.State.DOWN, null)
                            TunnelRuntime.emit(TunnelRuntime.currentState(applicationContext))
                            main.post { result.success(null) }
                        } catch (_: Exception) { TunnelRuntime.emit("failed"); main.post { result.error("stop_failed", "Não foi possível confirmar a desconexão.", null) } }
                    }
                }
                else -> result.notImplemented()
            }
        }
    }
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        if (requestCode == 4102) {
            permissionResult?.success(resultCode == Activity.RESULT_OK && VpnService.prepare(this) == null)
            permissionResult = null
        } else super.onActivityResult(requestCode, resultCode, data)
    }
    override fun onDestroy() {
        listening = false; main.removeCallbacks(poll); sink = null; TunnelRuntime.observer = null
        permissionResult?.success(false); permissionResult = null
        super.onDestroy()
    }
}
