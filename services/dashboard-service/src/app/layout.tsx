import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'AIPX - AI-Powered Investment Platform',
  description: 'Intelligent trading system powered by artificial intelligence',
  keywords: ['AI', 'trading', 'investment', 'cryptocurrency', 'stock'],
  authors: [{ name: 'AIPX Team' }],
  viewport: 'width=device-width, initial-scale=1',
  themeColor: '#0a0a0a',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="ko" className="dark">
      <body className={inter.className}>
        {children}
      </body>
    </html>
  )
}
