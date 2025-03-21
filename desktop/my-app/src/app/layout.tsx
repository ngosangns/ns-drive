import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { AppProvider } from "@/lib/app-context";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "NS Drive",
  description: "File synchronization tool",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        <link
          href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500&display=swap"
          rel="stylesheet"
        />
        <link
          href="https://fonts.googleapis.com/icon?family=Material+Icons"
          rel="stylesheet"
        />
      </head>
      <body
        className={`${inter.className} mat-app-background mat-typography bg-gray-900 text-white`}
      >
        <AppProvider>{children}</AppProvider>
      </body>
    </html>
  );
}
