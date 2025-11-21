import Navbar from '@/components/Navbar'

/**
 * Dashboard layout component
 *
 * Features:
 * - Persistent navigation bar
 * - Responsive layout
 * - Protected route wrapper
 *
 * Usage:
 * Wraps all dashboard pages
 */
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen bg-gray-950">
      <Navbar />
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  )
}
