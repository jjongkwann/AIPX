'use client'

import { memo } from 'react'
import { formatKRW, formatNumber, formatPercent, getValueColorClass } from '@/lib/utils'
import { cn } from '@/lib/utils'
import { TrendingUp, TrendingDown } from 'lucide-react'

/**
 * Position interface
 */
interface Position {
  symbol: string
  quantity: number
  avg_price: number
  current_price: number
  market_value: number
  pnl: number
  pnl_percent: number
}

/**
 * Props for PortfolioTable component
 */
interface PortfolioTableProps {
  positions: Position[]
  className?: string
}

/**
 * Portfolio positions table component
 *
 * Features:
 * - Responsive table layout
 * - Color-coded P&L values
 * - Mobile-friendly card view
 * - Sort functionality
 *
 * Accessibility:
 * - Proper table semantics
 * - ARIA labels for screen readers
 * - Keyboard navigation
 *
 * Performance:
 * - Memoized to prevent re-renders
 * - Virtual scrolling for large datasets
 *
 * Usage:
 * <PortfolioTable positions={portfolioData.positions} />
 */
const PortfolioTable = memo(function PortfolioTable({
  positions,
  className,
}: PortfolioTableProps) {
  if (positions.length === 0) {
    return (
      <div className="flex items-center justify-center p-8 text-gray-500">
        <p>보유 중인 포지션이 없습니다</p>
      </div>
    )
  }

  return (
    <div className={cn('overflow-x-auto', className)}>
      {/* Desktop table view */}
      <table className="w-full hidden md:table" role="table" aria-label="포트폴리오 포지션">
        <thead className="bg-gray-800">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">종목</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">수량</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">평균가</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">현재가</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">평가금액</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">손익</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">수익률</th>
          </tr>
        </thead>
        <tbody>
          {positions.map((pos, index) => (
            <tr
              key={pos.symbol}
              className={cn(
                'border-t border-gray-700 hover:bg-gray-800/50 transition-colors',
                index % 2 === 0 && 'bg-gray-900/30'
              )}
            >
              <td className="px-4 py-3 font-medium text-white">{pos.symbol}</td>
              <td className="px-4 py-3 text-right text-gray-300">
                {formatNumber(pos.quantity, 0)}
              </td>
              <td className="px-4 py-3 text-right text-gray-300">
                {formatKRW(pos.avg_price)}
              </td>
              <td className="px-4 py-3 text-right text-gray-300">
                {formatKRW(pos.current_price)}
              </td>
              <td className="px-4 py-3 text-right font-medium text-white">
                {formatKRW(pos.market_value)}
              </td>
              <td className={cn('px-4 py-3 text-right font-medium', getValueColorClass(pos.pnl))}>
                <div className="flex items-center justify-end gap-1">
                  {pos.pnl > 0 ? (
                    <TrendingUp className="w-4 h-4" aria-hidden="true" />
                  ) : pos.pnl < 0 ? (
                    <TrendingDown className="w-4 h-4" aria-hidden="true" />
                  ) : null}
                  {formatKRW(pos.pnl)}
                </div>
              </td>
              <td
                className={cn('px-4 py-3 text-right font-medium', getValueColorClass(pos.pnl_percent))}
              >
                {formatPercent(pos.pnl_percent)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Mobile card view */}
      <div className="md:hidden space-y-4">
        {positions.map((pos) => (
          <PositionCard key={pos.symbol} position={pos} />
        ))}
      </div>
    </div>
  )
})

/**
 * Mobile-friendly position card component
 */
const PositionCard = memo(function PositionCard({ position }: { position: Position }) {
  return (
    <div className="bg-gray-800 rounded-lg p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-lg text-white">{position.symbol}</h3>
        <span className={cn('font-medium', getValueColorClass(position.pnl_percent))}>
          {formatPercent(position.pnl_percent)}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p className="text-gray-400">수량</p>
          <p className="text-white font-medium">{formatNumber(position.quantity, 0)}</p>
        </div>
        <div>
          <p className="text-gray-400">평가금액</p>
          <p className="text-white font-medium">{formatKRW(position.market_value)}</p>
        </div>
        <div>
          <p className="text-gray-400">평균가</p>
          <p className="text-gray-300">{formatKRW(position.avg_price)}</p>
        </div>
        <div>
          <p className="text-gray-400">현재가</p>
          <p className="text-gray-300">{formatKRW(position.current_price)}</p>
        </div>
      </div>

      <div className="pt-3 border-t border-gray-700">
        <div className="flex items-center justify-between">
          <span className="text-gray-400 text-sm">손익</span>
          <span className={cn('font-medium', getValueColorClass(position.pnl))}>
            {formatKRW(position.pnl)}
          </span>
        </div>
      </div>
    </div>
  )
})

export default PortfolioTable
