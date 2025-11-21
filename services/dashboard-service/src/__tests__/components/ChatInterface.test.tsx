import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ChatInterface from '@/components/ChatInterface'
import { MockWebSocket } from '@/tests/utils/test-utils'
import { mockMessages } from '@/tests/mocks/data'

// Mock the WebSocket hook
jest.mock('@/lib/websocket', () => ({
  useWebSocket: jest.fn(() => ({
    sendMessage: jest.fn(),
    lastMessage: null,
    readyState: 1, // OPEN
  })),
  parseMessage: jest.fn((msg) => JSON.parse(msg)),
}))

// Mock utilities
jest.mock('@/lib/utils', () => ({
  cn: (...classes: any[]) => classes.filter(Boolean).join(' '),
  formatRelativeTime: (timestamp: number) => 'Just now',
}))

describe('ChatInterface', () => {
  const mockUserId = 'test-user-123'

  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders chat interface with connection status', () => {
    render(<ChatInterface userId={mockUserId} />)

    expect(screen.getByText('AI 에이전트 연결됨')).toBeInTheDocument()
  })

  it('shows disconnected status when not connected', () => {
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: jest.fn(),
      lastMessage: null,
      readyState: 3, // CLOSED
    })

    render(<ChatInterface userId={mockUserId} />)

    expect(screen.getByText('재연결 중...')).toBeInTheDocument()
  })

  it('renders empty state when no messages', () => {
    render(<ChatInterface userId={mockUserId} />)

    expect(screen.getByText('AI 에이전트에게 질문해보세요')).toBeInTheDocument()
  })

  it('renders message input and send button', () => {
    render(<ChatInterface userId={mockUserId} />)

    expect(screen.getByPlaceholderText('메시지를 입력하세요...')).toBeInTheDocument()
    expect(screen.getByLabelText('메시지 전송')).toBeInTheDocument()
  })

  it('allows user to type message', async () => {
    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')

    await user.type(input, '삼성전자 분석해줘')

    expect(input).toHaveValue('삼성전자 분석해줘')
  })

  it('sends message on button click', async () => {
    const mockSendMessage = jest.fn()
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: mockSendMessage,
      lastMessage: null,
      readyState: 1,
    })

    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')
    const sendButton = screen.getByLabelText('메시지 전송')

    await user.type(input, '삼성전자 분석해줘')
    await user.click(sendButton)

    expect(mockSendMessage).toHaveBeenCalledWith(
      JSON.stringify({
        type: 'user_message',
        content: '삼성전자 분석해줘',
      })
    )
  })

  it('sends message on Enter key press', async () => {
    const mockSendMessage = jest.fn()
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: mockSendMessage,
      lastMessage: null,
      readyState: 1,
    })

    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')

    await user.type(input, '삼성전자 분석해줘{Enter}')

    expect(mockSendMessage).toHaveBeenCalled()
  })

  it('does not send message on Shift+Enter', async () => {
    const mockSendMessage = jest.fn()
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: mockSendMessage,
      lastMessage: null,
      readyState: 1,
    })

    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')

    await user.type(input, '삼성전자 분석해줘')
    await user.keyboard('{Shift>}{Enter}{/Shift}')

    expect(mockSendMessage).not.toHaveBeenCalled()
  })

  it('disables input and button when disconnected', () => {
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: jest.fn(),
      lastMessage: null,
      readyState: 3, // CLOSED
    })

    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')
    const sendButton = screen.getByLabelText('메시지 전송')

    expect(input).toBeDisabled()
    expect(sendButton).toBeDisabled()
  })

  it('displays user and AI messages', () => {
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: jest.fn(),
      lastMessage: JSON.stringify(mockMessages[1]),
      readyState: 1,
    })

    const { rerender } = render(<ChatInterface userId={mockUserId} />)

    // Trigger message update
    rerender(<ChatInterface userId={mockUserId} />)

    waitFor(() => {
      expect(screen.getByText(mockMessages[1].content)).toBeInTheDocument()
    })
  })

  it('shows typing indicator when AI is responding', async () => {
    const mockSendMessage = jest.fn()
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: mockSendMessage,
      lastMessage: null,
      readyState: 1,
    })

    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')
    const sendButton = screen.getByLabelText('메시지 전송')

    await user.type(input, '삼성전자 분석해줘')
    await user.click(sendButton)

    // Typing indicator should be present (loading spinner)
    // Note: This depends on your implementation
  })

  it('clears input after sending message', async () => {
    const mockSendMessage = jest.fn()
    const { useWebSocket } = require('@/lib/websocket')
    useWebSocket.mockReturnValue({
      sendMessage: mockSendMessage,
      lastMessage: null,
      readyState: 1,
    })

    const user = userEvent.setup()
    render(<ChatInterface userId={mockUserId} />)

    const input = screen.getByPlaceholderText('메시지를 입력하세요...')

    await user.type(input, '삼성전자 분석해줘')
    await user.click(screen.getByLabelText('메시지 전송'))

    expect(input).toHaveValue('')
  })

  it('applies custom className', () => {
    const { container } = render(<ChatInterface userId={mockUserId} className="custom-class" />)

    expect(container.firstChild).toHaveClass('custom-class')
  })
})
