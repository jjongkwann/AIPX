'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { getBacktests } from '@/lib/api'
import { formatKRW, formatPercent, getValueColorClass, formatRelativeTime } from '@/lib/utils'
import { Loader2, Plus, BarChart3 } from 'lucide-react'

/**
 * Backtest list page component
 *
 * Features:
 * - List of backtest results
 * - Quick metrics view
 * - Navigation to detailed results
 *
 * Usage:
 * Navigate to /dashboard/backtest
 */
export default function BacktestPage() {
  const [isLoading, setIsLoading] = useState(true)
  const [backtests, setBacktests] = useState<any[]>([])

  useEffect(() => {
    async function fetchBacktests() {
      try {
        const data = await getBacktests()
        setBacktests(data)
      } catch (error) {
        console.error('Failed to fetch backtests:', error)
      } finally {
        setIsLoading(false)
      }
    }

    fetchBacktests()
  }, [])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-200px)]">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">백테스트</h1>
          <p className="text-gray-400">전략 성과를 분석하고 검증하세요</p>
        </div>
        <Button className="gap-2">
          <Plus className="w-4 h-4" />
          새 백테스트
        </Button>
      </div>

      {/* Backtest List */}
      {backtests.length === 0 ? (
        <Card className="bg-gray-900 border-gray-800">
          <CardContent className="py-12 text-center">
            <BarChart3 className="w-12 h-12 text-gray-600 mx-auto mb-4" />
            <p className="text-gray-500 mb-4">아직 백테스트 결과가 없습니다</p>
            <Button>첫 백테스트 실행하기</Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {backtests.map((backtest) => (
            <BacktestCard key={backtest.id} backtest={backtest} />
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * Backtest card component
 */
function BacktestCard({ backtest }: { backtest: any }) {
  const totalReturn = (backtest.final_capital - backtest.initial_capital) / backtest.initial_capital

  return (
    <Link href={`/dashboard/backtest/${backtest.id}`}>
      <Card className="bg-gray-900 border-gray-800 hover:border-gray-700 transition-colors cursor-pointer">
        <CardHeader>
          <div className="flex items-start justify-between">
            <div>
              <CardTitle className="text-xl mb-2">{backtest.strategy_name}</CardTitle>
              <p className="text-sm text-gray-400">
                {backtest.start_date} ~ {backtest.end_date}
              </p>
            </div>
            <Badge variant={totalReturn > 0 ? 'default' : 'destructive'}>
              {formatPercent(totalReturn * 100)}
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricItem label="초기 자본" value={formatKRW(backtest.initial_capital)} />
            <MetricItem label="최종 자본" value={formatKRW(backtest.final_capital)} />
            <MetricItem
              label="수익"
              value={formatKRW(backtest.final_capital - backtest.initial_capital)}
              valueColor={getValueColorClass(backtest.final_capital - backtest.initial_capital)}
            />
            <MetricItem
              label="실행 시간"
              value={formatRelativeTime(backtest.created_at)}
            />
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

/**
 * Metric item component
 */
function MetricItem({
  label,
  value,
  valueColor = 'text-white',
}: {
  label: string
  value: string
  valueColor?: string
}) {
  return (
    <div>
      <p className="text-xs text-gray-400 mb-1">{label}</p>
      <p className={`font-medium ${valueColor}`}>{value}</p>
    </div>
  )
}
