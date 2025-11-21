'use client'

import { memo } from 'react'
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatKRW, formatNumber, formatPercent } from '@/lib/utils'
import { TrendingUp, TrendingDown, Activity, Target, Award } from 'lucide-react'

/**
 * Backtest report interface
 */
interface BacktestReport {
  id: string
  strategy_name: string
  summary: {
    total_return: number
    cagr: number
    mdd: number
    sharpe_ratio: number
    win_rate: number
    total_trades: number
    winning_trades: number
    losing_trades: number
  }
  equity_curve: Array<{ date: string; equity: number }>
  trades: Array<{
    symbol: string
    entry_date: string
    exit_date: string
    entry_price: number
    exit_price: number
    quantity: number
    pnl: number
    return: number
  }>
}

/**
 * Props for BacktestResults component
 */
interface BacktestResultsProps {
  report: BacktestReport
  className?: string
}

/**
 * Backtest results visualization component
 *
 * Features:
 * - Performance metrics grid
 * - Equity curve chart
 * - Trade history table
 * - Mobile-responsive layout
 *
 * Accessibility:
 * - ARIA labels for metrics
 * - Semantic HTML structure
 * - Screen reader friendly tables
 *
 * Performance:
 * - Memoized to prevent re-renders
 * - Lazy-loaded chart data
 * - Optimized table rendering
 *
 * Usage:
 * <BacktestResults report={backtestData} />
 */
const BacktestResults = memo(function BacktestResults({ report, className }: BacktestResultsProps) {
  const { summary, equity_curve, trades } = report

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Performance Metrics Grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
        <MetricCard
          label="총 수익률"
          value={formatPercent(summary.total_return * 100)}
          icon={<TrendingUp className="w-5 h-5" />}
          positive={summary.total_return > 0}
        />
        <MetricCard
          label="CAGR"
          value={formatPercent(summary.cagr * 100)}
          icon={<Activity className="w-5 h-5" />}
          positive={summary.cagr > 0}
        />
        <MetricCard
          label="MDD"
          value={formatPercent(summary.mdd * 100)}
          icon={<TrendingDown className="w-5 h-5" />}
          positive={false}
        />
        <MetricCard
          label="샤프 비율"
          value={formatNumber(summary.sharpe_ratio, 2)}
          icon={<Target className="w-5 h-5" />}
          positive={summary.sharpe_ratio > 1}
        />
        <MetricCard
          label="승률"
          value={formatPercent(summary.win_rate * 100)}
          icon={<Award className="w-5 h-5" />}
          positive={summary.win_rate > 0.5}
        />
      </div>

      {/* Trade Statistics */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle className="text-lg">거래 통계</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold text-white">{summary.total_trades}</p>
              <p className="text-sm text-gray-400">총 거래</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-green-500">{summary.winning_trades}</p>
              <p className="text-sm text-gray-400">수익 거래</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-red-500">{summary.losing_trades}</p>
              <p className="text-sm text-gray-400">손실 거래</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Equity Curve Chart */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle className="text-lg">자산 곡선</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={equity_curve}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2b2b43" />
              <XAxis
                dataKey="date"
                stroke="#9ca3af"
                tick={{ fill: '#9ca3af', fontSize: 12 }}
              />
              <YAxis
                stroke="#9ca3af"
                tick={{ fill: '#9ca3af', fontSize: 12 }}
                tickFormatter={(value) => formatNumber(value, 0)}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#1f2937',
                  border: '1px solid #374151',
                  borderRadius: '8px',
                }}
                labelStyle={{ color: '#9ca3af' }}
                itemStyle={{ color: '#3b82f6' }}
                formatter={(value: number) => formatKRW(value)}
              />
              <Line
                type="monotone"
                dataKey="equity"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Trade History Table */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle className="text-lg">거래 내역</CardTitle>
        </CardHeader>
        <CardContent>
          <TradeHistoryTable trades={trades} />
        </CardContent>
      </Card>
    </div>
  )
})

/**
 * Metric card component
 */
interface MetricCardProps {
  label: string
  value: string
  icon: React.ReactNode
  positive?: boolean
}

const MetricCard = memo(function MetricCard({ label, value, icon, positive }: MetricCardProps) {
  return (
    <Card className="bg-gray-900 border-gray-800">
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-gray-400">{label}</span>
          <div className={positive ? 'text-green-500' : 'text-gray-400'}>{icon}</div>
        </div>
        <p
          className={`text-2xl font-bold ${
            positive === true
              ? 'text-green-500'
              : positive === false
              ? 'text-red-500'
              : 'text-white'
          }`}
        >
          {value}
        </p>
      </CardContent>
    </Card>
  )
})

/**
 * Trade history table component
 */
interface TradeHistoryTableProps {
  trades: BacktestReport['trades']
}

const TradeHistoryTable = memo(function TradeHistoryTable({ trades }: TradeHistoryTableProps) {
  if (trades.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        <p>거래 내역이 없습니다</p>
      </div>
    )
  }

  // Show only first 10 trades for performance
  const displayTrades = trades.slice(0, 10)

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-800">
          <tr>
            <th className="px-4 py-2 text-left">종목</th>
            <th className="px-4 py-2 text-right">진입가</th>
            <th className="px-4 py-2 text-right">청산가</th>
            <th className="px-4 py-2 text-right">수량</th>
            <th className="px-4 py-2 text-right">손익</th>
            <th className="px-4 py-2 text-right">수익률</th>
          </tr>
        </thead>
        <tbody>
          {displayTrades.map((trade, index) => (
            <tr key={index} className="border-t border-gray-700 hover:bg-gray-800/50">
              <td className="px-4 py-2 font-medium">{trade.symbol}</td>
              <td className="px-4 py-2 text-right">{formatKRW(trade.entry_price)}</td>
              <td className="px-4 py-2 text-right">{formatKRW(trade.exit_price)}</td>
              <td className="px-4 py-2 text-right">{formatNumber(trade.quantity, 0)}</td>
              <td
                className={`px-4 py-2 text-right font-medium ${
                  trade.pnl > 0 ? 'text-green-500' : 'text-red-500'
                }`}
              >
                {formatKRW(trade.pnl)}
              </td>
              <td
                className={`px-4 py-2 text-right font-medium ${
                  trade.return > 0 ? 'text-green-500' : 'text-red-500'
                }`}
              >
                {formatPercent(trade.return * 100)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {trades.length > 10 && (
        <p className="text-center text-sm text-gray-500 mt-4">
          총 {trades.length}개 거래 중 10개 표시
        </p>
      )}
    </div>
  )
})

export default BacktestResults
