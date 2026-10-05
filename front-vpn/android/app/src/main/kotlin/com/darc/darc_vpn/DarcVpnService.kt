package com.darc.darc_vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import com.wireguard.android.backend.GoBackend

class DarcVpnService : GoBackend.VpnService() {
    override fun onCreate() {
        super.onCreate()
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(NotificationChannel("darc_vpn", "Conexão VPN", NotificationManager.IMPORTANCE_LOW))
        }
        val launch = PendingIntent.getActivity(this, 0, Intent(this, MainActivity::class.java), PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) Notification.Builder(this, "darc_vpn") else Notification.Builder(this)
        val notification = builder.setContentTitle("Darc VPN")
            .setContentText("Controle a conexão pelo aplicativo.")
            .setSmallIcon(android.R.drawable.ic_lock_lock).setContentIntent(launch).setOngoing(true).build()
        if (Build.VERSION.SDK_INT >= 34) startForeground(4101, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_SYSTEM_EXEMPTED)
        else startForeground(4101, notification)
    }
}
