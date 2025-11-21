import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Navbar from '@/components/Navbar'

// Mock next/navigation
const mockPush = jest.fn()
const mockPathname = '/dashboard'

jest.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
  useRouter: () => ({
    push: mockPush,
  }),
}))

// Mock logout function
jest.mock('@/lib/api', () => ({
  logout: jest.fn(),
}))

// Mock utils
jest.mock('@/lib/utils', () => ({
  cn: (...classes: any[]) => classes.filter(Boolean).join(' '),
}))

describe('Navbar', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders logo and navigation items', () => {
    render(<Navbar />)

    expect(screen.getByText('AIPX')).toBeInTheDocument()
    expect(screen.getByText('대시보드')).toBeInTheDocument()
    expect(screen.getByText('AI 채팅')).toBeInTheDocument()
    expect(screen.getByText('포트폴리오')).toBeInTheDocument()
    expect(screen.getByText('백테스트')).toBeInTheDocument()
    expect(screen.getByText('전략')).toBeInTheDocument()
  })

  it('highlights active navigation item', () => {
    render(<Navbar />)

    const dashboardLink = screen.getAllByText('대시보드')[0].closest('a')
    expect(dashboardLink).toHaveAttribute('href', '/dashboard')
  })

  it('renders logout button', () => {
    render(<Navbar />)

    const logoutButtons = screen.getAllByText('로그아웃')
    expect(logoutButtons.length).toBeGreaterThan(0)
  })

  it('calls logout function when logout button is clicked', async () => {
    const { logout } = require('@/lib/api')
    const user = userEvent.setup()

    render(<Navbar />)

    const logoutButton = screen.getAllByLabelText('로그아웃')[0]
    await user.click(logoutButton)

    expect(logout).toHaveBeenCalled()
  })

  it('renders mobile menu button on mobile', () => {
    render(<Navbar />)

    const menuButton = screen.getByLabelText('메뉴 열기')
    expect(menuButton).toBeInTheDocument()
  })

  it('toggles mobile menu on button click', async () => {
    const user = userEvent.setup()
    render(<Navbar />)

    const menuButton = screen.getByLabelText('메뉴 열기')

    // Mobile menu should not be visible initially
    expect(screen.queryAllByRole('link', { name: '대시보드' })).toHaveLength(1)

    // Click to open menu
    await user.click(menuButton)

    // Now there should be more links (desktop + mobile)
    await waitFor(() => {
      expect(screen.queryAllByRole('link', { name: '대시보드' }).length).toBeGreaterThan(1)
    })
  })

  it('closes mobile menu when a link is clicked', async () => {
    const user = userEvent.setup()
    render(<Navbar />)

    const menuButton = screen.getByLabelText('메뉴 열기')

    // Open mobile menu
    await user.click(menuButton)

    await waitFor(() => {
      const links = screen.queryAllByRole('link', { name: 'AI 채팅' })
      expect(links.length).toBeGreaterThan(1)
    })

    // Click a mobile menu link
    const mobileLinks = screen.getAllByRole('link', { name: 'AI 채팅' })
    await user.click(mobileLinks[mobileLinks.length - 1])

    // Menu should close (fewer links visible)
    await waitFor(() => {
      expect(screen.queryAllByRole('link', { name: 'AI 채팅' })).toHaveLength(1)
    })
  })

  it('has proper ARIA labels for navigation', () => {
    render(<Navbar />)

    const nav = screen.getByRole('navigation', { name: '주요 네비게이션' })
    expect(nav).toBeInTheDocument()
  })

  it('renders all navigation links with correct hrefs', () => {
    render(<Navbar />)

    const expectedLinks = [
      { text: '대시보드', href: '/dashboard' },
      { text: 'AI 채팅', href: '/dashboard/chat' },
      { text: '포트폴리오', href: '/dashboard/portfolio' },
      { text: '백테스트', href: '/dashboard/backtest' },
      { text: '전략', href: '/dashboard/strategies' },
    ]

    expectedLinks.forEach(({ text, href }) => {
      const link = screen.getAllByText(text)[0].closest('a')
      expect(link).toHaveAttribute('href', href)
    })
  })

  it('renders icons for each navigation item', () => {
    const { container } = render(<Navbar />)

    // Check that SVG icons are present
    const icons = container.querySelectorAll('svg')
    expect(icons.length).toBeGreaterThan(0)
  })

  it('applies active styles to current page', () => {
    const { container } = render(<Navbar />)

    // The dashboard button should have active styles
    const dashboardButton = screen.getAllByText('대시보드')[0].closest('button')
    expect(dashboardButton).toHaveClass('text-white', 'bg-gray-800')
  })
})
