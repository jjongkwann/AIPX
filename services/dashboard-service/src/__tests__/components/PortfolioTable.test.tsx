import React from 'react'
import { render, screen, within } from '@testing-library/react'
import PortfolioTable from '@/components/PortfolioTable'
import { mockPortfolio } from '@/tests/mocks/data'

// Mock utility functions
jest.mock('@/lib/utils', () => ({
  formatKRW: (value: number) => `₩${value.toLocaleString()}`,
  formatNumber: (value: number, decimals: number) => value.toFixed(decimals),
  formatPercent: (value: number) => `${value.toFixed(2)}%`,
  getValueColorClass: (value: number) => (value > 0 ? 'text-green-500' : value < 0 ? 'text-red-500' : 'text-gray-400'),
  cn: (...classes: any[]) => classes.filter(Boolean).join(' '),
}))

describe('PortfolioTable', () => {
  const positions = mockPortfolio.positions

  it('renders table with positions data', () => {
    render(<PortfolioTable positions={positions} />)

    // Check table headers
    expect(screen.getByText('종목')).toBeInTheDocument()
    expect(screen.getByText('수량')).toBeInTheDocument()
    expect(screen.getByText('평균가')).toBeInTheDocument()
    expect(screen.getByText('현재가')).toBeInTheDocument()
    expect(screen.getByText('평가금액')).toBeInTheDocument()
    expect(screen.getByText('손익')).toBeInTheDocument()
    expect(screen.getByText('수익률')).toBeInTheDocument()
  })

  it('displays all position data', () => {
    render(<PortfolioTable positions={positions} />)

    // Check first position
    expect(screen.getByText('005930')).toBeInTheDocument()
    expect(screen.getByText('100.00')).toBeInTheDocument()
    expect(screen.getByText('₩70,000')).toBeInTheDocument()
    expect(screen.getByText('₩71,500')).toBeInTheDocument()
  })

  it('handles empty positions array', () => {
    render(<PortfolioTable positions={[]} />)

    expect(screen.getByText('보유 중인 포지션이 없습니다')).toBeInTheDocument()
  })

  it('applies green color to positive P&L', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const greenTexts = container.querySelectorAll('.text-green-500')
    expect(greenTexts.length).toBeGreaterThan(0)
  })

  it('applies red color to negative P&L', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const redTexts = container.querySelectorAll('.text-red-500')
    expect(redTexts.length).toBeGreaterThan(0)
  })

  it('shows trending icons for P&L values', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const icons = container.querySelectorAll('svg')
    expect(icons.length).toBeGreaterThan(0)
  })

  it('renders with proper ARIA label', () => {
    render(<PortfolioTable positions={positions} />)

    const table = screen.getByRole('table', { name: '포트폴리오 포지션' })
    expect(table).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(<PortfolioTable positions={positions} className="custom-class" />)

    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('renders mobile card view for each position', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    // Mobile cards should be present (they're hidden on desktop)
    const mobileCards = container.querySelectorAll('.md\\:hidden .bg-gray-800')
    expect(mobileCards.length).toBe(positions.length)
  })

  it('displays correct number formatting', () => {
    render(<PortfolioTable positions={positions} />)

    // Check that numbers are formatted with commas
    expect(screen.getByText('₩70,000')).toBeInTheDocument()
    expect(screen.getByText('₩7,150,000')).toBeInTheDocument()
  })

  it('displays percentage values correctly', () => {
    render(<PortfolioTable positions={positions} />)

    expect(screen.getByText('2.14%')).toBeInTheDocument()
    expect(screen.getByText('-1.54%')).toBeInTheDocument()
    expect(screen.getByText('5.00%')).toBeInTheDocument()
  })

  it('shows hover effect on table rows', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const rows = container.querySelectorAll('tbody tr')
    rows.forEach(row => {
      expect(row).toHaveClass('hover:bg-gray-800/50')
    })
  })

  it('applies alternating row colors', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const rows = container.querySelectorAll('tbody tr')
    expect(rows[0]).toHaveClass('bg-gray-900/30')
  })

  it('renders all position symbols', () => {
    render(<PortfolioTable positions={positions} />)

    positions.forEach(pos => {
      const symbols = screen.getAllByText(pos.symbol)
      expect(symbols.length).toBeGreaterThan(0)
    })
  })

  it('mobile card shows key metrics', () => {
    const { container } = render(<PortfolioTable positions={positions} />)

    const mobileSection = container.querySelector('.md\\:hidden')
    expect(mobileSection).toBeInTheDocument()

    if (mobileSection) {
      expect(within(mobileSection as HTMLElement).getByText('수량')).toBeInTheDocument()
      expect(within(mobileSection as HTMLElement).getByText('평가금액')).toBeInTheDocument()
      expect(within(mobileSection as HTMLElement).getByText('평균가')).toBeInTheDocument()
      expect(within(mobileSection as HTMLElement).getByText('현재가')).toBeInTheDocument()
      expect(within(mobileSection as HTMLElement).getByText('손익')).toBeInTheDocument()
    }
  })
})
