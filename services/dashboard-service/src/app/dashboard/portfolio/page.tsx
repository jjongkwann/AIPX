'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import PortfolioTable from '@/components/PortfolioTable'
import { getPortfolio } from '@/lib/api'
import { formatKRW, formatPercent, getValueColorClass } from '@/lib/utils'
import { Loader2, PieChart, TrendingUp, DollarSign } from 'lucide-react'

/**
 * Portfolio page component
 *
 * Features:
 * - Portfolio summary
 * - Position table
 * - Performance metrics
 *
 * Usage:
 * Navigate to /dashboard/portfolio
 */
export default function PortfolioPage() {
  const [isLoading, setIsLoading] = useState(true)
  const [portfolioData, setPortfolioData] = useState<any>(null)

  useEffect(() => {
    async function fetchPortfolio() {
      try {
        // Mock user ID - in real app, get from auth context
        const userId = 'user-123'
        const data = await getPortfolio(userId)
        setPortfolioData(data)
      } catch (error) {
        console.error('Failed to fetch portfolio:', error)
      } finally {
        setIsLoading(false)
      }
    }

    fetchPortfolio()
  }, [])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-200px)]">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  if (!portfolioData) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">포트폴리오 데이터를 불러올 수 없습니다</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-white mb-2">포트폴리오</h1>
        <p className="text-gray-400">보유 자산과 포지션을 관리하세요</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <SummaryCard
          label="총 자산"
          value={formatKRW(portfolioData.total_value)}
          icon={<DollarSign className="w-5 h-5" />}
        />
        <SummaryCard
          label="투자금"
          value={formatKRW(portfolioData.invested)}
          icon={<PieChart className="w-5 h-5" />}
        />
        <SummaryCard
          label="현금"
          value={formatKRW(portfolioData.cash)}
          icon={<DollarSign className="w-5 h-5" />}
        />
        <SummaryCard
          label="총 손익"
          value={formatKRW(portfolioData.total_pnl)}
          valueColor={getValueColorClass(portfolioData.total_pnl)}
          subValue={formatPercent(portfolioData.total_pnl_percent)}
          icon={<TrendingUp className="w-5 h-5" />}
        />
      </div>

      {/* Performance Overview */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle>수익률</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div
              className={`text-4xl font-bold ${getValueColorClass(portfolioData.total_pnl_percent)}`}
            >
              {formatPercent(portfolioData.total_pnl_percent)}
            </div>
            <div className="text-gray-400">
              <p className="text-sm">투자 대비 수익</p>
              <p className={`text-lg font-semibold ${getValueColorClass(portfolioData.total_pnl)}`}>
                {formatKRW(portfolioData.total_pnl)}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Positions Table */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle>보유 포지션</CardTitle>
        </CardHeader>
        <CardContent>
          <PortfolioTable positions={portfolioData.positions || []} />
        </CardContent>
      </Card>

      {/* Asset Allocation */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle>자산 배분</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <AllocationBar
              label="투자 자산"
              value={portfolioData.invested}
              total={portfolioData.total_value}
              color="blue"
            />
            <AllocationBar
              label="현금"
              value={portfolioData.cash}
              total={portfolioData.total_value}
              color="green"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Summary card component
 */
function SummaryCard({
  label,
  value,
  valueColor = 'text-white',
  subValue,
  icon,
}: {
  label: string
  value: string
  valueColor?: string
  subValue?: string
  icon: React.ReactNode
}) {
  return (
    <Card className="bg-gray-900 border-gray-800">
      <CardContent className="p-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-400">{label}</span>
          <div className="text-gray-400">{icon}</div>
        </div>
        <p className={`text-2xl font-bold ${valueColor} mb-1`}>{value}</p>
        {subValue && <p className={`text-sm ${valueColor}`}>{subValue}</p>}
      </CardContent>
    </Card>
  )
}

/**
 * Allocation bar component
 */
function AllocationBar({
  label,
  value,
  total,
  color,
}: {
  label: string
  value: number
  total: number
  color: 'blue' | 'green'
}) {
  const percentage = (value / total) * 100
  const colorClasses = {
    blue: 'bg-blue-600',
    green: 'bg-green-600',
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-gray-400">{label}</span>
        <span className="text-sm text-white">
          {formatKRW(value)} ({percentage.toFixed(1)}%)
        </span>
      </div>
      <div className="w-full bg-gray-800 rounded-full h-2">
        <div
          className={`${colorClasses[color]} h-2 rounded-full transition-all`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  )
}
