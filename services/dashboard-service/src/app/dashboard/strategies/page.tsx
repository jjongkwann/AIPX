'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { getStrategies } from '@/lib/api'
import { formatPercent } from '@/lib/utils'
import { Loader2, Plus, Lightbulb, TrendingUp } from 'lucide-react'

/**
 * Strategies page component
 *
 * Features:
 * - List of available strategies
 * - Strategy details
 * - Activation/deactivation
 *
 * Usage:
 * Navigate to /dashboard/strategies
 */
export default function StrategiesPage() {
  const [isLoading, setIsLoading] = useState(true)
  const [strategies, setStrategies] = useState<any[]>([])

  useEffect(() => {
    async function fetchStrategies() {
      try {
        const data = await getStrategies()
        setStrategies(data)
      } catch (error) {
        console.error('Failed to fetch strategies:', error)
      } finally {
        setIsLoading(false)
      }
    }

    fetchStrategies()
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
          <h1 className="text-3xl font-bold text-white mb-2">전략</h1>
          <p className="text-gray-400">투자 전략을 관리하고 활성화하세요</p>
        </div>
        <Button className="gap-2">
          <Plus className="w-4 h-4" />
          새 전략
        </Button>
      </div>

      {/* Strategy Grid */}
      {strategies.length === 0 ? (
        <Card className="bg-gray-900 border-gray-800">
          <CardContent className="py-12 text-center">
            <Lightbulb className="w-12 h-12 text-gray-600 mx-auto mb-4" />
            <p className="text-gray-500 mb-4">아직 등록된 전략이 없습니다</p>
            <Button>첫 전략 만들기</Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid md:grid-cols-2 gap-6">
          {strategies.map((strategy) => (
            <StrategyCard key={strategy.id} strategy={strategy} />
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * Strategy card component
 */
function StrategyCard({ strategy }: { strategy: any }) {
  return (
    <Card className="bg-gray-900 border-gray-800">
      <CardHeader>
        <div className="flex items-start justify-between mb-2">
          <div>
            <CardTitle className="text-xl mb-2">{strategy.name}</CardTitle>
            <Badge variant={strategy.status === 'active' ? 'default' : 'secondary'}>
              {strategy.status === 'active' ? '활성' : '비활성'}
            </Badge>
          </div>
          <Badge variant="outline">{strategy.type}</Badge>
        </div>
        <CardDescription>{strategy.description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Performance Metrics */}
        <div className="grid grid-cols-3 gap-4 py-4 border-t border-gray-800">
          <div className="text-center">
            <p className="text-xs text-gray-400 mb-1">수익률</p>
            <p className="font-semibold text-white">
              {formatPercent(strategy.performance.total_return * 100)}
            </p>
          </div>
          <div className="text-center">
            <p className="text-xs text-gray-400 mb-1">샤프 비율</p>
            <p className="font-semibold text-white">
              {strategy.performance.sharpe_ratio.toFixed(2)}
            </p>
          </div>
          <div className="text-center">
            <p className="text-xs text-gray-400 mb-1">승률</p>
            <p className="font-semibold text-white">
              {formatPercent(strategy.performance.win_rate * 100)}
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-2">
          <Button
            variant={strategy.status === 'active' ? 'destructive' : 'default'}
            className="flex-1"
          >
            {strategy.status === 'active' ? '비활성화' : '활성화'}
          </Button>
          <Button variant="outline" className="flex-1 gap-2">
            <TrendingUp className="w-4 h-4" />
            백테스트
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
