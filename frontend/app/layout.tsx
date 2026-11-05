import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "HADESGATE™ // Tactical Data Anonymization Gateway",
  description:
    "High-performance Go AI Gateway for deterministic real-time PII & secret anonymization.",
};

const themeInitScript = `
(function() {
  try {
    var saved = localStorage.getItem('hadesgate_theme');
    if (saved === 'light') {
      document.documentElement.classList.remove('dark');
    } else {
      document.documentElement.classList.add('dark');
    }
  } catch (e) {
    document.documentElement.classList.add('dark');
  }
})();
`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR" className="dark" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Archivo+Black&family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-screen bg-background text-foreground antialiased selection:bg-purple-600 selection:text-white">
        {/* Global CRT Scanline & Mechanical Grain Overlay */}
        <div
          aria-hidden="true"
          className="brutalist-scanlines pointer-events-none fixed inset-0 z-50 opacity-70"
        />
        <svg
          aria-hidden="true"
          className="pointer-events-none fixed inset-0 z-50 h-full w-full opacity-[0.025]"
        >
          <filter id="hadesgate-noise">
            <feTurbulence
              type="fractalNoise"
              baseFrequency="0.85"
              numOctaves="2"
              stitchTiles="stitch"
            />
          </filter>
          <rect width="100%" height="100%" filter="url(#hadesgate-noise)" />
        </svg>

        {children}
      </body>
    </html>
  );
}
