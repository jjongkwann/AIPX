import React, { ReactElement } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { SessionProvider } from 'next-auth/react'

/**
 * Custom render function with providers
 */
interface AllProvidersProps {
  children: React.ReactNode
}

const AllProviders = ({ children }: AllProvidersProps) => {
  const mockSession = {
    user: {
      id: 'test-user-id',
      email: 'test@example.com',
      name: 'Test User',
    },
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  }

  return (
    <SessionProvider session={mockSession}>
      {children}
    </SessionProvider>
  )
}

const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => render(ui, { wrapper: AllProviders, ...options })

export * from '@testing-library/react'
export { customRender as render }

/**
 * Helper to wait for async updates
 */
export const waitForNextUpdate = () =>
  new Promise((resolve) => setTimeout(resolve, 0))

/**
 * Mock WebSocket
 */
export class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  public readyState = MockWebSocket.CONNECTING
  public onopen: ((event: Event) => void) | null = null
  public onmessage: ((event: MessageEvent) => void) | null = null
  public onerror: ((event: Event) => void) | null = null
  public onclose: ((event: CloseEvent) => void) | null = null

  constructor(public url: string) {
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    }, 0)
  }

  send(data: string) {
    // Mock send implementation
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close'))
    }
  }

  // Helper method to simulate receiving a message
  simulateMessage(data: any) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }))
    }
  }
}

/**
 * Create mock candlestick data
 */
export const createMockCandlestickData = (count: number = 100) => {
  const data = []
  let basePrice = 70000
  const startTime = Date.now() - count * 24 * 60 * 60 * 1000

  for (let i = 0; i < count; i++) {
    const time = (startTime + i * 24 * 60 * 60 * 1000) / 1000
    const open = basePrice + (Math.random() - 0.5) * 1000
    const close = open + (Math.random() - 0.5) * 2000
    const high = Math.max(open, close) + Math.random() * 1000
    const low = Math.min(open, close) - Math.random() * 1000

    data.push({ time, open, high, low, close })
    basePrice = close
  }

  return data
}
