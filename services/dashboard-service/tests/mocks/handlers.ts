import { http, HttpResponse } from 'msw'
import {
  mockPortfolio,
  mockBacktestReport,
  mockBacktestList,
  mockUser,
  mockStrategies,
} from './data'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8000'

export const handlers = [
  // Auth endpoints
  http.post(`${API_BASE}/api/v1/auth/login`, async ({ request }) => {
    const body = await request.json() as any

    if (body.email === 'test@example.com' && body.password === 'password123') {
      return HttpResponse.json({
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        user: mockUser,
      })
    }

    return HttpResponse.json(
      { detail: 'Invalid credentials' },
      { status: 401 }
    )
  }),

  http.post(`${API_BASE}/api/v1/auth/signup`, async ({ request }) => {
    const body = await request.json() as any

    return HttpResponse.json({
      user: {
        id: 'new-user-id',
        email: body.email,
        name: body.name,
      },
    }, { status: 201 })
  }),

  http.post(`${API_BASE}/api/v1/auth/logout`, () => {
    return HttpResponse.json({ message: 'Logged out successfully' })
  }),

  http.post(`${API_BASE}/api/v1/auth/refresh`, () => {
    return HttpResponse.json({
      access_token: 'new-mock-access-token',
    })
  }),

  // Portfolio endpoints
  http.get(`${API_BASE}/api/v1/portfolio`, ({ request }) => {
    const authHeader = request.headers.get('Authorization')

    if (!authHeader) {
      return HttpResponse.json(
        { detail: 'Not authenticated' },
        { status: 401 }
      )
    }

    return HttpResponse.json(mockPortfolio)
  }),

  http.get(`${API_BASE}/api/v1/portfolio/positions`, () => {
    return HttpResponse.json(mockPortfolio.positions)
  }),

  // Backtest endpoints
  http.get(`${API_BASE}/api/v1/backtests`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') || '1')
    const limit = parseInt(url.searchParams.get('limit') || '10')

    return HttpResponse.json({
      items: mockBacktestList,
      total: mockBacktestList.length,
      page,
      limit,
    })
  }),

  http.get(`${API_BASE}/api/v1/backtests/:id`, ({ params }) => {
    return HttpResponse.json(mockBacktestReport)
  }),

  http.post(`${API_BASE}/api/v1/backtests`, async ({ request }) => {
    const body = await request.json() as any

    return HttpResponse.json({
      id: 'new-backtest-id',
      strategy_name: body.strategy_name,
      status: 'queued',
    }, { status: 201 })
  }),

  http.delete(`${API_BASE}/api/v1/backtests/:id`, () => {
    return HttpResponse.json({ message: 'Backtest deleted' })
  }),

  // Strategy endpoints
  http.get(`${API_BASE}/api/v1/strategies`, () => {
    return HttpResponse.json(mockStrategies)
  }),

  http.get(`${API_BASE}/api/v1/strategies/:id`, ({ params }) => {
    return HttpResponse.json(mockStrategies[0])
  }),

  http.post(`${API_BASE}/api/v1/strategies`, async ({ request }) => {
    const body = await request.json() as any

    return HttpResponse.json({
      id: 'new-strategy-id',
      ...body,
      created_at: new Date().toISOString(),
    }, { status: 201 })
  }),

  // Market data endpoints
  http.get(`${API_BASE}/api/v1/market/quote/:symbol`, ({ params }) => {
    return HttpResponse.json({
      symbol: params.symbol,
      price: 71500,
      change: 1500,
      change_percent: 2.14,
      volume: 10000000,
      timestamp: new Date().toISOString(),
    })
  }),

  // Error simulation endpoints
  http.get(`${API_BASE}/api/v1/error/500`, () => {
    return HttpResponse.json(
      { detail: 'Internal server error' },
      { status: 500 }
    )
  }),

  http.get(`${API_BASE}/api/v1/error/403`, () => {
    return HttpResponse.json(
      { detail: 'Forbidden' },
      { status: 403 }
    )
  }),
]
