import type { Metadata } from "next";
import type { ReactNode } from "react";
import { JetBrains_Mono, Schibsted_Grotesk, Source_Serif_4 } from "next/font/google";
import Script from "next/script";
import { SiteFooter } from "@/components/site-footer";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "SourceInk | Publish from GitHub",
    template: "%s | SourceInk",
  },
  description: "Publish technical writing from the Git repository you already own.",
  icons: { icon: "/mark.svg" },
};

const sans = Schibsted_Grotesk({ subsets: ["latin"], variable: "--font-sans", display: "swap" });
const serif = Source_Serif_4({ subsets: ["latin"], variable: "--font-serif", display: "swap", style: ["normal", "italic"] });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono", display: "swap" });

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="en" data-theme="dark" className={`${sans.variable} ${serif.variable} ${mono.variable}`} suppressHydrationWarning>
      <head>
        <Script
          id="sourceink-theme"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{
            __html: "try{document.documentElement.dataset.theme=localStorage.getItem('sourceink-theme')==='light'?'light':'dark'}catch{}",
          }}
        />
      </head>
      <body>
        <a className="skip-link" href="#main-content">
          Skip to content
        </a>
        {children}
        <SiteFooter />
      </body>
    </html>
  );
}
