'use client'

import { useEffect, useRef, useState, useCallback } from 'react'

/**
 * WebSocket connection states
 */
export enum ReadyState {
  CONNECTING = 0,
  OPEN = 1,
  CLOSING = 2,
  CLOSED = 3,
}

/**
 * WebSocket hook options
 */
interface UseWebSocketOptions {
  onOpen?: (event: Event) => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
  reconnect?: boolean
  reconnectAttempts?: number
  reconnectInterval?: number
}

/**
 * Custom hook for WebSocket connection with auto-reconnect
 *
 * Usage:
 * const { sendMessage, lastMessage, readyState } = useWebSocket(url)
 *
 * Performance: Uses refs to avoid unnecessary re-renders
 * Accessibility: Provides clear connection state feedback
 */
export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const {
    onOpen,
    onClose,
    onError,
    reconnect = true,
    reconnectAttempts = 5,
    reconnectInterval = 3000,
  } = options

  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null)
  const [readyState, setReadyState] = useState<ReadyState>(ReadyState.CONNECTING)

  const ws = useRef<WebSocket | null>(null)
  const reconnectCount = useRef(0)
  const reconnectTimeout = useRef<NodeJS.Timeout>()
  const unmounted = useRef(false)

  /**
   * Connect to WebSocket server
   */
  const connect = useCallback(() => {
    if (unmounted.current) return

    try {
      ws.current = new WebSocket(url)

      ws.current.onopen = (event) => {
        setReadyState(ReadyState.OPEN)
        reconnectCount.current = 0
        onOpen?.(event)
      }

      ws.current.onclose = (event) => {
        setReadyState(ReadyState.CLOSED)
        onClose?.(event)

        // Auto-reconnect logic
        if (reconnect && reconnectCount.current < reconnectAttempts && !unmounted.current) {
          reconnectTimeout.current = setTimeout(() => {
            reconnectCount.current++
            connect()
          }, reconnectInterval)
        }
      }

      ws.current.onerror = (event) => {
        onError?.(event)
      }

      ws.current.onmessage = (event) => {
        setLastMessage(event)
      }
    } catch (error) {
      console.error('WebSocket connection error:', error)
    }
  }, [url, reconnect, reconnectAttempts, reconnectInterval, onOpen, onClose, onError])

  /**
   * Send message through WebSocket
   */
  const sendMessage = useCallback(
    (data: string | ArrayBufferLike | Blob | ArrayBufferView) => {
      if (ws.current?.readyState === ReadyState.OPEN) {
        ws.current.send(data)
      } else {
        console.warn('WebSocket is not connected')
      }
    },
    []
  )

  /**
   * Close WebSocket connection
   */
  const disconnect = useCallback(() => {
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current)
    }
    if (ws.current) {
      ws.current.close()
    }
  }, [])

  /**
   * Initialize connection on mount, cleanup on unmount
   */
  useEffect(() => {
    connect()

    return () => {
      unmounted.current = true
      disconnect()
    }
  }, [connect, disconnect])

  return {
    sendMessage,
    lastMessage,
    readyState,
    disconnect,
  }
}

/**
 * Parse JSON message safely
 */
export function parseMessage<T = any>(message: MessageEvent | null): T | null {
  if (!message) return null

  try {
    return JSON.parse(message.data) as T
  } catch (error) {
    console.error('Failed to parse message:', error)
    return null
  }
}
