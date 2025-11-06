import type { Metadata, Viewport } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from '@/components/providers'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: {
    default: 'Immobilier SaaS - Find Your Perfect Property',
    template: '%s | Immobilier SaaS'
  },
  description: 'Modern real-estate platform for renting apartments, houses, and villas in Senegal',
  keywords: ['real estate', 'rental', 'apartment', 'house', 'senegal', 'dakar'],
  authors: [{ name: 'Immobilier SaaS Team' }],
  creator: 'Immobilier SaaS',
  metadataBase: new URL(process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'),
  openGraph: {
    type: 'website',
    locale: 'fr_SN',
    url: process.env.NEXT_PUBLIC_APP_URL,
    title: 'Immobilier SaaS - Find Your Perfect Property',
    description: 'Modern real-estate platform for renting properties',
    siteName: 'Immobilier SaaS',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Immobilier SaaS',
    description: 'Find your perfect property',
    creator: '@immobiliersaas',
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
    icon: '/icons/icon-192x192.png',
    apple: '/icons/icon-192x192.png',
  },
}

export const viewport: Viewport = {
  themeColor: '#ffffff',
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="fr" suppressHydrationWarning>
      <body className={inter.className}>
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  )
}
