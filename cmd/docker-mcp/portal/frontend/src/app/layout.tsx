import type { Metadata } from 'next';
import { Inter, JetBrains_Mono } from 'next/font/google';
import { AuthContextProvider } from '@/contexts/AuthContext';
import AuthProvider from '@/providers/AuthProvider';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-inter',
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-jetbrains-mono',
});

export const metadata: Metadata = {
  title: {
    default: 'MCP Portal',
    template: '%s | MCP Portal',
  },
  description:
    'A comprehensive web interface for managing Model Context Protocol (MCP) servers with Docker integration.',
  keywords: [
    'MCP',
    'Model Context Protocol',
    'Docker',
    'Server Management',
    'Container Management',
    'Development Tools',
  ],
  authors: [{ name: 'Docker Inc.' }],
  creator: 'Docker Inc.',
  publisher: 'Docker Inc.',
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  metadataBase: new URL(
    process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
  ),
  openGraph: {
    title: 'MCP Portal',
    description:
      'A comprehensive web interface for managing Model Context Protocol (MCP) servers with Docker integration.',
    url: '/',
    siteName: 'MCP Portal',
    locale: 'en_US',
    type: 'website',
    images: [
      {
        url: '/og-image.png',
        width: 1200,
        height: 630,
        alt: 'MCP Portal - Server Management Dashboard',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'MCP Portal',
    description:
      'A comprehensive web interface for managing Model Context Protocol (MCP) servers with Docker integration.',
    images: ['/twitter-image.png'],
    creator: '@docker',
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
  manifest: '/manifest.json',
  icons: {
    icon: '/favicon.ico',
    shortcut: '/favicon-16x16.png',
    apple: '/apple-touch-icon.png',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${inter.variable} ${jetbrainsMono.variable} font-sans antialiased`}
        suppressHydrationWarning
      >
        <AuthProvider>
          <AuthContextProvider>
            <div className="relative flex min-h-screen flex-col">
              <main className="flex-1">{children}</main>
            </div>
          </AuthContextProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
