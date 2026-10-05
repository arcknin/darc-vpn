import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "VPN Access · Administração",
  description: "Dashboard administrativo local do VPN Access",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
