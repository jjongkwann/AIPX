import React from 'react'
import { render, screen } from '@testing-library/react'
import BacktestResults from '@/components/BacktestResults'
import { mockBacktestReport } from '@/tests/mocks/data'

// Mock recharts to avoid canvas issues in tests
jest.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
  LineChart: ({ children }: any) => <div>{children}</div>,
  Line: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  CartesianGrid: () => null,
}))

// Mock utility functions
jest.mock('@/lib/utils', () => ({
  formatKRW: (value: number) => `₩${value.toLocaleString()}`,
  formatNumber: (value: number, decimals: number) => value.toFixed(decimals),
  formatPercent: (value: number) => `${value.toFixed(2)}%`,
}))

describe('BacktestResults', () => {
  it('renders performance metrics', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    expect(screen.getByText('총 수익률')).toBeInTheDocument()
    expect(screen.getByText('CAGR')).toBeInTheDocument()
    expect(screen.getByText('MDD')).toBeInTheDocument()
    expect(screen.getByText('샤프 비율')).toBeInTheDocument()
    expect(screen.getByText('승률')).toBeInTheDocument()
  })

  it('displays metric values correctly', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    expect(screen.getByText('15.00%')).toBeInTheDocument() // total_return
    expect(screen.getByText('12.00%')).toBeInTheDocument() // cagr
    expect(screen.getByText('-8.00%')).toBeInTheDocument() // mdd
    expect(screen.getByText('1.50')).toBeInTheDocument() // sharpe_ratio
    expect(screen.getByText('58.00%')).toBeInTheDocument() // win_rate
  })

  it('applies correct color coding for positive metrics', () => {
    const { container } = render(<BacktestResults report={mockBacktestReport} />)

    // Positive metrics should have green color
    const positiveMetrics = container.querySelectorAll('.text-green-500')
    expect(positiveMetrics.length).toBeGreaterThan(0)
  })

  it('applies correct color coding for negative metrics', () => {
    const { container } = render(<BacktestResults report={mockBacktestReport} />)

    // MDD should be red
    const negativeMetrics = container.querySelectorAll('.text-red-500')
    expect(negativeMetrics.length).toBeGreaterThan(0)
  })

  it('renders trade statistics section', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    expect(screen.getByText('거래 통계')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument() // total_trades
    expect(screen.getByText('58')).toBeInTheDocument() // winning_trades
    expect(screen.getByText('42')).toBeInTheDocument() // losing_trades
  })

  it('renders equity curve chart', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    expect(screen.getByText('자산 곡선')).toBeInTheDocument()
  })

  it('renders trade history table', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    expect(screen.getByText('거래 내역')).toBeInTheDocument()
    expect(screen.getByText('종목')).toBeInTheDocument()
    expect(screen.getByText('진입가')).toBeInTheDocument()
    expect(screen.getByText('청산가')).toBeInTheDocument()
    expect(screen.getByText('수량')).toBeInTheDocument()
    expect(screen.getByText('손익')).toBeInTheDocument()
    expect(screen.getByText('수익률')).toBeInTheDocument()
  })

  it('displays trade data in table', () => {
    render(<BacktestResults report={mockBacktestReport} />)

    // Check first trade
    expect(screen.getByText('005930')).toBeInTheDocument()
    expect(screen.getByText('₩68,000')).toBeInTheDocument()
    expect(screen.getByText('₩71,000')).toBeInTheDocument()
  })

  it('handles empty trade history', () => {
    const emptyReport = {
      ...mockBacktestReport,
      trades: [],
    }

    render(<BacktestResults report={emptyReport} />)

    expect(screen.getByText('거래 내역이 없습니다')).toBeInTheDocument()
  })

  it('limits trade display to 10 items', () => {
    const manyTradesReport = {
      ...mockBacktestReport,
      trades: Array(25).fill(mockBacktestReport.trades[0]).map((trade, i) => ({
        ...trade,
        symbol: `00${i}930`,
      })),
    }

    render(<BacktestResults report={manyTradesReport} />)

    expect(screen.getByText('총 25개 거래 중 10개 표시')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <BacktestResults report={mockBacktestReport} className="custom-class" />
    )

    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('renders icons for metrics', () => {
    const { container } = render(<BacktestResults report={mockBacktestReport} />)

    const icons = container.querySelectorAll('svg')
    expect(icons.length).toBeGreaterThan(0)
  })

  it('colors winning trades green and losing trades red', () => {
    const { container } = render(<BacktestResults report={mockBacktestReport} />)

    // Check for profit/loss color coding in table
    const greenTexts = container.querySelectorAll('.text-green-500')
    const redTexts = container.querySelectorAll('.text-red-500')

    expect(greenTexts.length).toBeGreaterThan(0)
    expect(redTexts.length).toBeGreaterThan(0)
  })

  it('renders responsive grid layout', () => {
    const { container } = render(<BacktestResults report={mockBacktestReport} />)

    const grid = container.querySelector('.grid')
    expect(grid).toHaveClass('grid-cols-2', 'md:grid-cols-3', 'lg:grid-cols-5')
  })
})
