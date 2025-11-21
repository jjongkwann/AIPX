'use client'

import ChatInterface from '@/components/ChatInterface'
import { Card } from '@/components/ui/card'

/**
 * Chat page component
 *
 * Features:
 * - Real-time chat with AI agent
 * - WebSocket connection
 * - Message history
 *
 * Usage:
 * Navigate to /dashboard/chat
 */
export default function ChatPage() {
  // Mock user ID - in real app, get from auth context
  const userId = 'user-123'

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-white mb-2">AI 채팅</h1>
        <p className="text-gray-400">AI 에이전트와 대화하고 투자 전략을 논의하세요</p>
      </div>

      {/* Chat Interface */}
      <Card className="bg-gray-900 border-gray-800 overflow-hidden">
        <div className="h-[calc(100vh-300px)]">
          <ChatInterface userId={userId} />
        </div>
      </Card>

      {/* Help Section */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <SuggestionCard
          title="시장 분석"
          suggestion="현재 비트코인 시장 상황을 분석해주세요"
        />
        <SuggestionCard
          title="전략 제안"
          suggestion="단기 트레이딩 전략을 추천해주세요"
        />
        <SuggestionCard
          title="포트폴리오 리뷰"
          suggestion="내 포트폴리오를 검토해주세요"
        />
      </div>
    </div>
  )
}

/**
 * Suggestion card component
 */
function SuggestionCard({ title, suggestion }: { title: string; suggestion: string }) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-gray-600 transition-colors cursor-pointer">
      <h3 className="font-medium text-white mb-2">{title}</h3>
      <p className="text-sm text-gray-400">{suggestion}</p>
    </div>
  )
}
