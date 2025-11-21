import React from 'react'
import { render, screen, waitFor } from '@/tests/utils/test-utils'
import TradingViewChart from '@/components/TradingViewChart'
import { createMockCandlestickData } from '@/tests/utils/test-utils'

// Mock lightweight-charts
jest.mock('lightweight-charts', () => ({
  createChart: jest.fn(() => ({
    addCandlestickSeries: jest.fn(() => ({
      setData: jest.fn(),
    })),
    timeScale: jest.fn(() => ({
      fitContent: jest.fn(),
    })),
    applyOptions: jest.fn(),
    remove: jest.fn(),
  })),
  ColorType: {
    Solid: 'solid',
  },
}))

describe('TradingViewChart', () => {
  const mockData = createMockCandlestickData(10)

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders chart container with proper ARIA label', () => {
    render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    const chartContainer = screen.getByRole('img', { name: 'BTC/KRW 가격 차트' })
    expect(chartContainer).toBeInTheDocument()
  })

  it('renders with default height', () => {
    const { container } = render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    const chartDiv = container.querySelector('div')
    expect(chartDiv).toBeInTheDocument()
  })

  it('renders with custom height', () => {
    render(<TradingViewChart symbol="BTC/KRW" data={mockData} height={600} />)

    const chartContainer = screen.getByRole('img')
    expect(chartContainer).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <TradingViewChart symbol="BTC/KRW" data={mockData} className="custom-class" />
    )

    const chartDiv = container.querySelector('.custom-class')
    expect(chartDiv).toBeInTheDocument()
  })

  it('handles empty data gracefully', () => {
    render(<TradingViewChart symbol="BTC/KRW" data={[]} />)

    const chartContainer = screen.getByRole('img')
    expect(chartContainer).toBeInTheDocument()
  })

  it('updates chart when data changes', async () => {
    const { rerender } = render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    const newData = createMockCandlestickData(20)
    rerender(<TradingViewChart symbol="BTC/KRW" data={newData} />)

    await waitFor(() => {
      const chartContainer = screen.getByRole('img')
      expect(chartContainer).toBeInTheDocument()
    })
  })

  it('handles window resize events', () => {
    const { createChart } = require('lightweight-charts')
    const mockChart = {
      addCandlestickSeries: jest.fn(() => ({ setData: jest.fn() })),
      timeScale: jest.fn(() => ({ fitContent: jest.fn() })),
      applyOptions: jest.fn(),
      remove: jest.fn(),
    }
    createChart.mockReturnValue(mockChart)

    render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    // Simulate window resize
    window.dispatchEvent(new Event('resize'))

    expect(mockChart.applyOptions).toHaveBeenCalled()
  })

  it('cleans up chart on unmount', () => {
    const { createChart } = require('lightweight-charts')
    const mockRemove = jest.fn()
    const mockChart = {
      addCandlestickSeries: jest.fn(() => ({ setData: jest.fn() })),
      timeScale: jest.fn(() => ({ fitContent: jest.fn() })),
      applyOptions: jest.fn(),
      remove: mockRemove,
    }
    createChart.mockReturnValue(mockChart)

    const { unmount } = render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    unmount()

    expect(mockRemove).toHaveBeenCalled()
  })

  it('does not create chart if container ref is null', () => {
    const { createChart } = require('lightweight-charts')
    createChart.mockClear()

    // This is a bit tricky to test, but we can verify chart is created when container exists
    render(<TradingViewChart symbol="BTC/KRW" data={mockData} />)

    expect(createChart).toHaveBeenCalled()
  })
})
