/**
 * Mock data for testing
 */

export const mockPortfolio = {
  positions: [
    {
      symbol: '005930',
      name: '삼성전자',
      quantity: 100,
      avg_price: 70000,
      current_price: 71500,
      market_value: 7150000,
      pnl: 150000,
      pnl_percent: 2.14,
    },
    {
      symbol: '000660',
      name: 'SK하이닉스',
      quantity: 50,
      avg_price: 130000,
      current_price: 128000,
      market_value: 6400000,
      pnl: -100000,
      pnl_percent: -1.54,
    },
    {
      symbol: '035420',
      name: 'NAVER',
      quantity: 20,
      avg_price: 200000,
      current_price: 210000,
      market_value: 4200000,
      pnl: 200000,
      pnl_percent: 5.0,
    },
  ],
  summary: {
    total_equity: 20000000,
    cash: 2250000,
    total_pnl: 250000,
    total_pnl_percent: 1.27,
  },
}

export const mockBacktestReport = {
  id: 'test-backtest-123',
  strategy_name: 'Momentum Strategy',
  start_date: '2023-01-01',
  end_date: '2023-12-31',
  initial_capital: 10000000,
  summary: {
    total_return: 0.15,
    cagr: 0.12,
    mdd: -0.08,
    sharpe_ratio: 1.5,
    win_rate: 0.58,
    total_trades: 100,
    winning_trades: 58,
    losing_trades: 42,
  },
  equity_curve: [
    { date: '2023-01', equity: 10000000 },
    { date: '2023-02', equity: 10200000 },
    { date: '2023-03', equity: 10150000 },
    { date: '2023-04', equity: 10500000 },
    { date: '2023-05', equity: 10450000 },
    { date: '2023-06', equity: 10800000 },
    { date: '2023-07', equity: 10750000 },
    { date: '2023-08', equity: 11000000 },
    { date: '2023-09', equity: 11100000 },
    { date: '2023-10', equity: 11200000 },
    { date: '2023-11', equity: 11350000 },
    { date: '2023-12', equity: 11500000 },
  ],
  trades: [
    {
      symbol: '005930',
      entry_date: '2023-01-15',
      exit_date: '2023-02-10',
      entry_price: 68000,
      exit_price: 71000,
      quantity: 100,
      pnl: 300000,
      return: 0.0441,
    },
    {
      symbol: '000660',
      entry_date: '2023-02-20',
      exit_date: '2023-03-15',
      entry_price: 135000,
      exit_price: 132000,
      quantity: 50,
      pnl: -150000,
      return: -0.0222,
    },
    {
      symbol: '035420',
      entry_date: '2023-03-25',
      exit_date: '2023-04-20',
      entry_price: 190000,
      exit_price: 205000,
      quantity: 30,
      pnl: 450000,
      return: 0.0789,
    },
  ],
}

export const mockBacktestList = [
  {
    id: 'backtest-1',
    strategy_name: 'Momentum Strategy',
    created_at: '2023-12-01T10:00:00Z',
    status: 'completed',
    total_return: 0.15,
  },
  {
    id: 'backtest-2',
    strategy_name: 'Mean Reversion Strategy',
    created_at: '2023-12-02T10:00:00Z',
    status: 'completed',
    total_return: 0.08,
  },
  {
    id: 'backtest-3',
    strategy_name: 'Trend Following Strategy',
    created_at: '2023-12-03T10:00:00Z',
    status: 'running',
    total_return: null,
  },
]

export const mockUser = {
  id: 'user-123',
  email: 'test@example.com',
  name: 'Test User',
}

export const mockMessages = [
  {
    role: 'user' as const,
    content: '삼성전자 주가 분석해줘',
    timestamp: Date.now() - 60000,
  },
  {
    role: 'assistant' as const,
    content: '삼성전자의 현재 주가는 71,500원입니다. 최근 반도체 시장 회복세로 상승 추세를 보이고 있습니다.',
    timestamp: Date.now() - 30000,
  },
]

export const mockCandlestickData = [
  { time: 1640995200, open: 70000, high: 72000, low: 69000, close: 71500 },
  { time: 1641081600, open: 71500, high: 73000, low: 71000, close: 72500 },
  { time: 1641168000, open: 72500, high: 74000, low: 72000, close: 73000 },
  { time: 1641254400, open: 73000, high: 74500, low: 72500, close: 74000 },
  { time: 1641340800, open: 74000, high: 75000, low: 73500, close: 74500 },
]

export const mockStrategies = [
  {
    id: 'strategy-1',
    name: 'Momentum Strategy',
    description: '모멘텀 기반 매매 전략',
    type: 'momentum',
    status: 'active',
    created_at: '2023-11-01T10:00:00Z',
  },
  {
    id: 'strategy-2',
    name: 'Mean Reversion Strategy',
    description: '평균 회귀 전략',
    type: 'mean_reversion',
    status: 'inactive',
    created_at: '2023-11-15T10:00:00Z',
  },
]
