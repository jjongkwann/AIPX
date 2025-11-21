import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import LoginPage from '@/app/(auth)/login/page'
import { server } from '@/tests/mocks/server'
import { http, HttpResponse } from 'msw'

// Mock next/navigation
const mockPush = jest.fn()
const mockRouter = { push: mockPush }

jest.mock('next/navigation', () => ({
  useRouter: () => mockRouter,
}))

// Mock API
jest.mock('@/lib/api', () => ({
  login: jest.fn(),
}))

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8000'

describe('LoginPage', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockPush.mockClear()
  })

  it('renders login form', () => {
    render(<LoginPage />)

    expect(screen.getByText('로그인')).toBeInTheDocument()
    expect(screen.getByLabelText('이메일 주소')).toBeInTheDocument()
    expect(screen.getByLabelText('비밀번호')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '로그인' })).toBeInTheDocument()
  })

  it('renders AIPX logo', () => {
    render(<LoginPage />)

    expect(screen.getByText('AIPX')).toBeInTheDocument()
  })

  it('renders signup link', () => {
    render(<LoginPage />)

    const signupLink = screen.getByText('회원가입')
    expect(signupLink).toBeInTheDocument()
    expect(signupLink.closest('a')).toHaveAttribute('href', '/signup')
  })

  it('allows user to type email and password', async () => {
    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')

    expect(emailInput).toHaveValue('test@example.com')
    expect(passwordInput).toHaveValue('password123')
  })

  it('validates required fields', async () => {
    const user = userEvent.setup()
    render(<LoginPage />)

    const submitButton = screen.getByRole('button', { name: '로그인' })
    await user.click(submitButton)

    // Browser will validate required fields
    const emailInput = screen.getByLabelText('이메일 주소')
    expect(emailInput).toBeRequired()
  })

  it('shows loading state during submission', async () => {
    const { login } = require('@/lib/api')
    login.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)))

    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')
    const submitButton = screen.getByRole('button', { name: '로그인' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    expect(screen.getByText('로그인 중...')).toBeInTheDocument()
    expect(submitButton).toBeDisabled()
  })

  it('redirects to dashboard on successful login', async () => {
    const { login } = require('@/lib/api')
    login.mockResolvedValue({ access_token: 'token' })

    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')
    const submitButton = screen.getByRole('button', { name: '로그인' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/dashboard')
    })
  })

  it('displays error message on failed login', async () => {
    const { login } = require('@/lib/api')
    login.mockRejectedValue({
      response: { data: { message: 'Invalid credentials' } }
    })

    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')
    const submitButton = screen.getByRole('button', { name: '로그인' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'wrongpassword')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid credentials')
    })
  })

  it('displays default error message when no message provided', async () => {
    const { login } = require('@/lib/api')
    login.mockRejectedValue(new Error('Network error'))

    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')
    const submitButton = screen.getByRole('button', { name: '로그인' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('로그인에 실패했습니다')
    })
  })

  it('clears error message on new submission', async () => {
    const { login } = require('@/lib/api')
    login.mockRejectedValueOnce({
      response: { data: { message: 'Invalid credentials' } }
    })

    const user = userEvent.setup()
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')
    const submitButton = screen.getByRole('button', { name: '로그인' })

    // First failed attempt
    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'wrongpassword')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })

    // Clear inputs
    await user.clear(emailInput)
    await user.clear(passwordInput)

    // Second attempt
    login.mockResolvedValue({ access_token: 'token' })
    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    // Error should be cleared
    await waitFor(() => {
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })
  })

  it('has proper accessibility attributes', () => {
    render(<LoginPage />)

    const emailInput = screen.getByLabelText('이메일 주소')
    const passwordInput = screen.getByLabelText('비밀번호')

    expect(emailInput).toHaveAttribute('type', 'email')
    expect(passwordInput).toHaveAttribute('type', 'password')
    expect(emailInput).toHaveAttribute('aria-label', '이메일 주소')
    expect(passwordInput).toHaveAttribute('aria-label', '비밀번호')
  })
})
