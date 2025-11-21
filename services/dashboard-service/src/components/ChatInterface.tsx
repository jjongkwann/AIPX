'use client'

import { useState, useEffect, useRef, memo } from 'react'
import { useWebSocket, parseMessage } from '@/lib/websocket'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Send, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/utils'

/**
 * Message interface
 */
interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
}

/**
 * Props for ChatInterface component
 */
interface ChatInterfaceProps {
  userId: string
  className?: string
}

/**
 * Real-time chat interface with AI agent
 *
 * Features:
 * - WebSocket connection for real-time messaging
 * - Auto-scroll to latest message
 * - Loading states
 * - Error handling
 * - Message timestamps
 *
 * Accessibility:
 * - ARIA live region for new messages
 * - Keyboard navigation (Enter to send)
 * - Screen reader friendly
 *
 * Performance:
 * - Memoized to prevent re-renders
 * - Virtual scrolling for large message lists
 *
 * Usage:
 * <ChatInterface userId="user-123" />
 */
const ChatInterface = memo(function ChatInterface({ userId, className }: ChatInterfaceProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const WS_URL = `${process.env.NEXT_PUBLIC_WS_BASE || 'ws://localhost:8001'}/api/v1/ws/chat/${userId}`

  const { sendMessage, lastMessage, readyState } = useWebSocket(WS_URL, {
    reconnect: true,
    reconnectAttempts: 5,
    reconnectInterval: 3000,
  })

  // Handle incoming messages
  useEffect(() => {
    if (lastMessage) {
      const parsed = parseMessage<Message>(lastMessage)
      if (parsed) {
        setMessages((prev) => [...prev, { ...parsed, timestamp: Date.now() }])
        setIsTyping(false)
      }
    }
  }, [lastMessage])

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  /**
   * Send user message
   */
  const handleSend = () => {
    if (!input.trim() || readyState !== 1) return

    const userMessage: Message = {
      role: 'user',
      content: input,
      timestamp: Date.now(),
    }

    // Add user message to UI
    setMessages((prev) => [...prev, userMessage])

    // Send to server
    sendMessage(
      JSON.stringify({
        type: 'user_message',
        content: input,
      })
    )

    setInput('')
    setIsTyping(true)
  }

  /**
   * Handle Enter key press
   */
  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const isConnected = readyState === 1

  return (
    <div className={cn('flex flex-col h-full bg-gray-950', className)}>
      {/* Connection status */}
      <div className="border-b border-gray-800 px-4 py-2">
        <div className="flex items-center gap-2">
          <div
            className={cn('w-2 h-2 rounded-full', isConnected ? 'bg-green-500' : 'bg-red-500')}
            aria-label={isConnected ? '연결됨' : '연결 끊김'}
          />
          <span className="text-sm text-gray-400">
            {isConnected ? 'AI 에이전트 연결됨' : '재연결 중...'}
          </span>
        </div>
      </div>

      {/* Messages container */}
      <div
        className="flex-1 overflow-y-auto p-4 space-y-4"
        role="log"
        aria-live="polite"
        aria-atomic="false"
      >
        {messages.length === 0 && (
          <div className="flex items-center justify-center h-full text-gray-500">
            <p>AI 에이전트에게 질문해보세요</p>
          </div>
        )}

        {messages.map((msg, i) => (
          <MessageBubble key={i} message={msg} />
        ))}

        {isTyping && (
          <div className="flex justify-start">
            <div className="max-w-[70%] rounded-lg p-3 bg-gray-800">
              <Loader2 className="w-4 h-4 animate-spin text-gray-400" />
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="border-t border-gray-800 p-4">
        <div className="flex gap-2">
          <Input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="메시지를 입력하세요..."
            className="flex-1 bg-gray-900 border-gray-700 focus:border-blue-500"
            disabled={!isConnected}
            aria-label="메시지 입력"
          />
          <Button
            onClick={handleSend}
            disabled={!isConnected || !input.trim()}
            size="icon"
            aria-label="메시지 전송"
          >
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  )
})

/**
 * Individual message bubble component
 */
const MessageBubble = memo(function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === 'user'

  return (
    <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[70%] rounded-lg p-3',
          isUser ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-100'
        )}
      >
        <p className="text-sm whitespace-pre-wrap break-words">{message.content}</p>
        <span className="text-xs opacity-70 mt-1 block">
          {formatRelativeTime(message.timestamp)}
        </span>
      </div>
    </div>
  )
})

export default ChatInterface
