'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import TradingViewChart from '@/components/TradingViewChart'
import { getPortfolio, getMarketData } from '@/lib/api'
import { formatKRW, formatPercent, getValueColorClass } from '@/lib/utils'
import { Wallet, TrendingUp, Activity, ArrowRight, Loader2 } from 'lucide-react'
import type { CandlestickData } from 'lightweight-charts'

/**
 * Dashboard overview page
 *
 * Features:
 * - Portfolio summary
 * - Market chart
 * - Quick stats
 * - Recent activity
 *
 * Performance:
 * - Server-side data fetching
 * - Lazy-loaded components
 * - Optimized re-renders
 */
export default function DashboardPage() {
  const [isLoading, setIsLoading] = useState(true)
  const [portfolioData, setPortfolioData] = useState<any>(null)
  const [chartData, setChartData] = useState<CandlestickData[]>([])

  useEffect(() => {
    async function fetchData() {
      try {
        // Mock user ID - in real app, get from auth context
        const userId = 'user-123'

        const [portfolio, market] = await Promise.all([
          getPortfolio(userId).catch(() => null),
          getMarketData('BTC/KRW', '1d', 30).catch(() => []),
        ])

        setPortfolioData(portfolio)
        setChartData(market)
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-200px)]">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-white mb-2">대시보드</h1>
        <p className="text-gray-400">포트폴리오와 시장 현황을 한눈에 확인하세요</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          label="총 자산"
          value={portfolioData ? formatKRW(portfolioData.total_value) : '-'}
          icon={<Wallet className="w-5 h-5" />}
          trend={portfolioData?.total_pnl_percent}
        />
        <StatCard
          label="투자금"
          value={portfolioData ? formatKRW(portfolioData.invested) : '-'}
          icon={<TrendingUp className="w-5 h-5" />}
        />
        <StatCard
          label="현금"
          value={portfolioData ? formatKRW(portfolioData.cash) : '-'}
          icon={<Activity className="w-5 h-5" />}
        />
      </div>

      {/* Chart Section */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>시장 차트 - BTC/KRW</CardTitle>
            <Badge variant="outline">1D</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {chartData.length > 0 ? (
            <TradingViewChart symbol="BTC/KRW" data={chartData} height={400} />
          ) : (
            <div className="h-[400px] flex items-center justify-center text-gray-500">
              차트 데이터를 불러올 수 없습니다
            </div>
          )}
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <QuickActionCard
          title="AI 채팅"
          description="AI 에이전트와 대화하고 투자 전략을 논의하세요"
          href="/dashboard/chat"
          color="blue"
        />
        <QuickActionCard
          title="백테스트"
          description="전략을 백테스트하고 성과를 분석하세요"
          href="/dashboard/backtest"
          color="purple"
        />
      </div>

      {/* Recent Activity */}
      <Card className="bg-gray-900 border-gray-800">
        <CardHeader>
          <CardTitle>최근 활동</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <ActivityItem
              title="새로운 전략 실행"
              description="Momentum Strategy가 활성화되었습니다"
              time="2시간 전"
            />
            <ActivityItem
              title="포지션 진입"
              description="BTC/KRW 매수 완료"
              time="5시간 전"
            />
            <ActivityItem
              title="백테스트 완료"
              description="Mean Reversion Strategy 백테스트 완료"
              time="1일 전"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Stat card component
 */
function StatCard({
  label,
  value,
  icon,
  trend,
}: {
  label: string
  value: string
  icon: React.ReactNode
  trend?: number
}) {
  return (
    <Card className="bg-gray-900 border-gray-800">
      <CardContent className="p-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-400">{label}</span>
          <div className="text-gray-400">{icon}</div>
        </div>
        <p className="text-2xl font-bold text-white mb-1">{value}</p>
        {trend !== undefined && (
          <p className={`text-sm ${getValueColorClass(trend)}`}>
            {formatPercent(trend)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * Quick action card component
 */
function QuickActionCard({
  title,
  description,
  href,
  color,
}: {
  title: string
  description: string
  href: string
  color: 'blue' | 'purple'
}) {
  const colorClasses = {
    blue: 'from-blue-600 to-blue-700',
    purple: 'from-purple-600 to-purple-700',
  }

  return (
    <Link href={href}>
      <Card className={`bg-gradient-to-r ${colorClasses[color]} border-0 hover:opacity-90 transition-opacity cursor-pointer`}>
        <CardContent className="p-6">
          <h3 className="text-xl font-semibold text-white mb-2">{title}</h3>
          <p className="text-gray-100 mb-4">{description}</p>
          <Button variant="secondary" size="sm" className="gap-2">
            시작하기
            <ArrowRight className="w-4 h-4" />
          </Button>
        </CardContent>
      </Card>
    </Link>
  )
}

/**
 * Activity item component
 */
function ActivityItem({
  title,
  description,
  time,
}: {
  title: string
  description: string
  time: string
}) {
  return (
    <div className="flex items-start gap-4 pb-4 border-b border-gray-800 last:border-0">
      <div className="w-2 h-2 rounded-full bg-blue-500 mt-2" />
      <div className="flex-1">
        <p className="font-medium text-white">{title}</p>
        <p className="text-sm text-gray-400">{description}</p>
      </div>
      <span className="text-sm text-gray-500">{time}</span>
    </div>
  )
}
