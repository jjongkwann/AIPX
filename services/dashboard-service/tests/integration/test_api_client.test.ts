/**
 * Integration tests for API client
 */

import { server } from '../mocks/server'
import { http, HttpResponse } from 'msw'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8000'

// Mock API client functions
const apiClient = {
  login: async (credentials: { email: string; password: string }) => {
    const response = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    })
    if (!response.ok) throw new Error('Login failed')
    return response.json()
  },

  getPortfolio: async (token: string) => {
    const response = await fetch(`${API_BASE}/api/v1/portfolio`, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
    if (!response.ok) throw new Error('Failed to fetch portfolio')
    return response.json()
  },

  getBacktests: async (token: string, page = 1, limit = 10) => {
    const response = await fetch(
      `${API_BASE}/api/v1/backtests?page=${page}&limit=${limit}`,
      {
        headers: { 'Authorization': `Bearer ${token}` },
      }
    )
    if (!response.ok) throw new Error('Failed to fetch backtests')
    return response.json()
  },

  createBacktest: async (token: string, data: any) => {
    const response = await fetch(`${API_BASE}/api/v1/backtests`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    })
    if (!response.ok) throw new Error('Failed to create backtest')
    return response.json()
  },
}

describe('API Client Integration Tests', () => {
  describe('Authentication', () => {
    it('successfully logs in with valid credentials', async () => {
      const result = await apiClient.login({
        email: 'test@example.com',
        password: 'password123',
      })

      expect(result).toHaveProperty('access_token')
      expect(result).toHaveProperty('refresh_token')
      expect(result.user.email).toBe('test@example.com')
    })

    it('fails to login with invalid credentials', async () => {
      await expect(
        apiClient.login({
          email: 'test@example.com',
          password: 'wrongpassword',
        })
      ).rejects.toThrow('Login failed')
    })
  })

  describe('Portfolio API', () => {
    const validToken = 'mock-access-token'

    it('fetches portfolio with valid token', async () => {
      const portfolio = await apiClient.getPortfolio(validToken)

      expect(portfolio).toHaveProperty('positions')
      expect(portfolio).toHaveProperty('summary')
      expect(Array.isArray(portfolio.positions)).toBe(true)
    })

    it('fails to fetch portfolio without token', async () => {
      await expect(apiClient.getPortfolio('')).rejects.toThrow()
    })

    it('handles 401 unauthorized error', async () => {
      server.use(
        http.get(`${API_BASE}/api/v1/portfolio`, () => {
          return HttpResponse.json(
            { detail: 'Not authenticated' },
            { status: 401 }
          )
        })
      )

      await expect(apiClient.getPortfolio('invalid-token')).rejects.toThrow()
    })
  })

  describe('Backtest API', () => {
    const validToken = 'mock-access-token'

    it('fetches backtest list with pagination', async () => {
      const result = await apiClient.getBacktests(validToken, 1, 10)

      expect(result).toHaveProperty('items')
      expect(result).toHaveProperty('total')
      expect(result).toHaveProperty('page')
      expect(result).toHaveProperty('limit')
      expect(Array.isArray(result.items)).toBe(true)
    })

    it('creates new backtest', async () => {
      const backtestData = {
        strategy_name: 'Test Strategy',
        start_date: '2023-01-01',
        end_date: '2023-12-31',
      }

      const result = await apiClient.createBacktest(validToken, backtestData)

      expect(result).toHaveProperty('id')
      expect(result.strategy_name).toBe('Test Strategy')
      expect(result.status).toBe('queued')
    })

    it('handles API errors gracefully', async () => {
      server.use(
        http.post(`${API_BASE}/api/v1/backtests`, () => {
          return HttpResponse.json(
            { detail: 'Invalid request' },
            { status: 400 }
          )
        })
      )

      await expect(
        apiClient.createBacktest(validToken, {})
      ).rejects.toThrow()
    })
  })

  describe('Error Handling', () => {
    it('handles 500 internal server error', async () => {
      server.use(
        http.get(`${API_BASE}/api/v1/error/500`, () => {
          return HttpResponse.json(
            { detail: 'Internal server error' },
            { status: 500 }
          )
        })
      )

      const response = await fetch(`${API_BASE}/api/v1/error/500`)
      expect(response.status).toBe(500)
    })

    it('handles 403 forbidden error', async () => {
      server.use(
        http.get(`${API_BASE}/api/v1/error/403`, () => {
          return HttpResponse.json(
            { detail: 'Forbidden' },
            { status: 403 }
          )
        })
      )

      const response = await fetch(`${API_BASE}/api/v1/error/403`)
      expect(response.status).toBe(403)
    })

    it('handles network errors', async () => {
      server.use(
        http.get(`${API_BASE}/api/v1/network-error`, () => {
          return HttpResponse.error()
        })
      )

      await expect(
        fetch(`${API_BASE}/api/v1/network-error`)
      ).rejects.toThrow()
    })
  })

  describe('Token Refresh', () => {
    it('refreshes access token', async () => {
      const response = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer refresh-token',
        },
      })

      const data = await response.json()
      expect(data).toHaveProperty('access_token')
      expect(data.access_token).toBe('new-mock-access-token')
    })
  })
})
