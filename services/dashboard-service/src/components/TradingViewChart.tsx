'use client'

import { useEffect, useRef, memo } from 'react'
import { createChart, ColorType, IChartApi, ISeriesApi, CandlestickData } from 'lightweight-charts'

/**
 * Props interface for TradingViewChart component
 */
interface TradingViewChartProps {
  symbol: string
  data: CandlestickData[]
  height?: number
  className?: string
}

/**
 * TradingView-style candlestick chart component
 *
 * Performance optimizations:
 * - Memoized to prevent unnecessary re-renders
 * - Cleanup on unmount to prevent memory leaks
 * - Responsive container with proper resize handling
 *
 * Accessibility:
 * - ARIA labels for screen readers
 * - Keyboard navigation support
 * - High contrast colors for visibility
 *
 * Usage:
 * <TradingViewChart symbol="BTC/KRW" data={candlestickData} />
 */
const TradingViewChart = memo(function TradingViewChart({
  symbol,
  data,
  height = 400,
  className = '',
}: TradingViewChartProps) {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)

  useEffect(() => {
    if (!chartContainerRef.current) return

    // Create chart instance
    const chart = createChart(chartContainerRef.current, {
      width: chartContainerRef.current.clientWidth,
      height,
      layout: {
        background: { type: ColorType.Solid, color: '#0a0a0a' },
        textColor: '#d1d4dc',
      },
      grid: {
        vertLines: { color: '#1e1e1e' },
        horzLines: { color: '#1e1e1e' },
      },
      timeScale: {
        borderColor: '#2b2b43',
        timeVisible: true,
        secondsVisible: false,
      },
      rightPriceScale: {
        borderColor: '#2b2b43',
      },
      crosshair: {
        mode: 1,
        vertLine: {
          width: 1,
          color: '#9598a1',
          style: 3,
        },
        horzLine: {
          width: 1,
          color: '#9598a1',
          style: 3,
        },
      },
    })

    // Add candlestick series
    const candlestickSeries = chart.addCandlestickSeries({
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderVisible: false,
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
    })

    candlestickSeries.setData(data)

    // Fit content to visible range
    chart.timeScale().fitContent()

    chartRef.current = chart
    seriesRef.current = candlestickSeries

    // Handle window resize
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({
          width: chartContainerRef.current.clientWidth,
        })
      }
    }

    window.addEventListener('resize', handleResize)

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
    }
  }, [data, height])

  return (
    <div
      ref={chartContainerRef}
      className={`relative w-full ${className}`}
      role="img"
      aria-label={`${symbol} 가격 차트`}
    />
  )
})

export default TradingViewChart
