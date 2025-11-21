'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import BacktestResults from '@/components/BacktestResults'
import { getBacktestReport } from '@/lib/api'
import { Loader2, ArrowLeft } from 'lucide-react'

/**
 * Backtest detail page component
 *
 * Features:
 * - Detailed backtest results
 * - Performance charts
 * - Trade history
 *
 * Usage:
 * Navigate to /dashboard/backtest/[id]
 */
export default function BacktestDetailPage() {
  const params = useParams()
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(true)
  const [report, setReport] = useState<any>(null)

  useEffect(() => {
    async function fetchReport() {
      try {
        const data = await getBacktestReport(params.id as string)
        setReport(data)
      } catch (error) {
        console.error('Failed to fetch backtest report:', error)
      } finally {
        setIsLoading(false)
      }
    }

    fetchReport()
  }, [params.id])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-200px)]">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  if (!report) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500 mb-4">백테스트 결과를 찾을 수 없습니다</p>
        <Button onClick={() => router.back()}>
          <ArrowLeft className="w-4 h-4 mr-2" />
          돌아가기
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.back()}
          aria-label="뒤로 가기"
        >
          <ArrowLeft className="w-5 h-5" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">{report.strategy_name}</h1>
          <p className="text-gray-400">백테스트 결과 상세 보기</p>
        </div>
      </div>

      {/* Results */}
      <BacktestResults report={report} />
    </div>
  )
}
