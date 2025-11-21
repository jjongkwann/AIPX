import axios, { AxiosError, AxiosInstance } from 'axios'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8000'

/**
 * Axios instance with default configuration
 */
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

/**
 * Request interceptor to add auth token
 */
apiClient.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

/**
 * Response interceptor for error handling
 */
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Redirect to login on unauthorized
      if (typeof window !== 'undefined') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

/**
 * Get token from localStorage
 */
function getToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem('auth_token')
}

/**
 * Store token in localStorage
 */
function setToken(token: string): void {
  if (typeof window !== 'undefined') {
    localStorage.setItem('auth_token', token)
  }
}

/**
 * Remove token from localStorage
 */
function removeToken(): void {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('auth_token')
  }
}

// ==================== Auth API ====================

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  token_type: string
  user: {
    id: string
    email: string
    username: string
  }
}

export async function login(data: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/api/v1/auth/login', data)
  setToken(response.data.access_token)
  return response.data
}

export interface SignupRequest {
  email: string
  password: string
  username: string
}

export async function signup(data: SignupRequest): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/api/v1/auth/signup', data)
  setToken(response.data.access_token)
  return response.data
}

export function logout(): void {
  removeToken()
  if (typeof window !== 'undefined') {
    window.location.href = '/login'
  }
}

// ==================== Portfolio API ====================

export interface Position {
  symbol: string
  quantity: number
  avg_price: number
  current_price: number
  market_value: number
  pnl: number
  pnl_percent: number
}

export interface PortfolioSummary {
  total_value: number
  cash: number
  invested: number
  total_pnl: number
  total_pnl_percent: number
  positions: Position[]
}

export async function getPortfolio(userId: string): Promise<PortfolioSummary> {
  const response = await apiClient.get<PortfolioSummary>(`/api/v1/portfolio/${userId}`)
  return response.data
}

// ==================== Backtest API ====================

export interface BacktestSummary {
  id: string
  strategy_name: string
  start_date: string
  end_date: string
  initial_capital: number
  final_capital: number
  total_return: number
  created_at: string
}

export interface BacktestReport {
  id: string
  strategy_name: string
  summary: {
    total_return: number
    cagr: number
    mdd: number
    sharpe_ratio: number
    win_rate: number
    total_trades: number
    winning_trades: number
    losing_trades: number
  }
  equity_curve: Array<{ date: string; equity: number }>
  trades: Array<{
    symbol: string
    entry_date: string
    exit_date: string
    entry_price: number
    exit_price: number
    quantity: number
    pnl: number
    return: number
  }>
  drawdowns: Array<{ date: string; drawdown: number }>
}

export async function getBacktests(): Promise<BacktestSummary[]> {
  const response = await apiClient.get<BacktestSummary[]>('/api/v1/backtests')
  return response.data
}

export async function getBacktestReport(id: string): Promise<BacktestReport> {
  const response = await apiClient.get<BacktestReport>(`/api/v1/backtests/${id}`)
  return response.data
}

export interface RunBacktestRequest {
  strategy_name: string
  symbols: string[]
  start_date: string
  end_date: string
  initial_capital: number
  parameters?: Record<string, any>
}

export async function runBacktest(data: RunBacktestRequest): Promise<{ id: string }> {
  const response = await apiClient.post<{ id: string }>('/api/v1/backtests/run', data)
  return response.data
}

// ==================== Strategy API ====================

export interface Strategy {
  id: string
  name: string
  description: string
  type: string
  parameters: Record<string, any>
  performance: {
    total_return: number
    sharpe_ratio: number
    win_rate: number
  }
  status: 'active' | 'inactive'
  created_at: string
}

export async function getStrategies(): Promise<Strategy[]> {
  const response = await apiClient.get<Strategy[]>('/api/v1/strategies')
  return response.data
}

export async function getStrategy(id: string): Promise<Strategy> {
  const response = await apiClient.get<Strategy>(`/api/v1/strategies/${id}`)
  return response.data
}

// ==================== Market Data API ====================

export interface CandlestickData {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume?: number
}

export async function getMarketData(
  symbol: string,
  interval: string = '1d',
  limit: number = 100
): Promise<CandlestickData[]> {
  const response = await apiClient.get<CandlestickData[]>('/api/v1/market/candles', {
    params: { symbol, interval, limit },
  })
  return response.data
}

export { getToken, setToken, removeToken }
