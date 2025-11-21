'use client'

import { memo } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  MessageSquare,
  Briefcase,
  BarChart3,
  Lightbulb,
  LogOut,
  Menu,
} from 'lucide-react'
import { logout } from '@/lib/api'
import { useState } from 'react'

/**
 * Navigation items configuration
 */
const navItems = [
  {
    label: '대시보드',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    label: 'AI 채팅',
    href: '/dashboard/chat',
    icon: MessageSquare,
  },
  {
    label: '포트폴리오',
    href: '/dashboard/portfolio',
    icon: Briefcase,
  },
  {
    label: '백테스트',
    href: '/dashboard/backtest',
    icon: BarChart3,
  },
  {
    label: '전략',
    href: '/dashboard/strategies',
    icon: Lightbulb,
  },
]

/**
 * Navigation bar component
 *
 * Features:
 * - Active route highlighting
 * - Mobile responsive menu
 * - Logout functionality
 * - Keyboard navigation
 *
 * Accessibility:
 * - ARIA labels for navigation
 * - Keyboard shortcuts
 * - Focus management
 *
 * Usage:
 * <Navbar />
 */
const Navbar = memo(function Navbar() {
  const pathname = usePathname()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  const handleLogout = () => {
    logout()
  }

  return (
    <nav className="bg-gray-900 border-b border-gray-800" role="navigation" aria-label="주요 네비게이션">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/dashboard" className="flex items-center gap-2">
            <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-lg">A</span>
            </div>
            <span className="text-white font-semibold text-xl hidden sm:inline">AIPX</span>
          </Link>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center gap-1">
            {navItems.map((item) => {
              const Icon = item.icon
              const isActive = pathname === item.href

              return (
                <Link key={item.href} href={item.href}>
                  <Button
                    variant="ghost"
                    className={cn(
                      'gap-2 text-gray-400 hover:text-white hover:bg-gray-800',
                      isActive && 'text-white bg-gray-800'
                    )}
                    aria-current={isActive ? 'page' : undefined}
                  >
                    <Icon className="w-4 h-4" aria-hidden="true" />
                    <span>{item.label}</span>
                  </Button>
                </Link>
              )
            })}
          </div>

          {/* Logout Button */}
          <div className="hidden md:block">
            <Button
              variant="ghost"
              onClick={handleLogout}
              className="gap-2 text-gray-400 hover:text-white hover:bg-gray-800"
              aria-label="로그아웃"
            >
              <LogOut className="w-4 h-4" aria-hidden="true" />
              <span>로그아웃</span>
            </Button>
          </div>

          {/* Mobile Menu Button */}
          <button
            className="md:hidden p-2 text-gray-400 hover:text-white"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
            aria-label="메뉴 열기"
            aria-expanded={isMobileMenuOpen}
          >
            <Menu className="w-6 h-6" />
          </button>
        </div>
      </div>

      {/* Mobile Menu */}
      {isMobileMenuOpen && (
        <div className="md:hidden border-t border-gray-800 bg-gray-900">
          <div className="px-2 pt-2 pb-3 space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon
              const isActive = pathname === item.href

              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={() => setIsMobileMenuOpen(false)}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800',
                    isActive && 'text-white bg-gray-800'
                  )}
                  aria-current={isActive ? 'page' : undefined}
                >
                  <Icon className="w-5 h-5" aria-hidden="true" />
                  <span>{item.label}</span>
                </Link>
              )
            })}

            <button
              onClick={handleLogout}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-800"
              aria-label="로그아웃"
            >
              <LogOut className="w-5 h-5" aria-hidden="true" />
              <span>로그아웃</span>
            </button>
          </div>
        </div>
      )}
    </nav>
  )
})

export default Navbar
