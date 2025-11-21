'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { signup } from '@/lib/api'
import { Loader2 } from 'lucide-react'

/**
 * Signup page component
 *
 * Features:
 * - User registration
 * - Form validation
 * - Error handling
 * - Loading states
 *
 * Accessibility:
 * - Form labels and ARIA attributes
 * - Keyboard navigation
 * - Error announcements
 *
 * Usage:
 * Navigate to /signup
 */
export default function SignupPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    // Validate passwords match
    if (password !== confirmPassword) {
      setError('비밀번호가 일치하지 않습니다')
      return
    }

    // Validate password length
    if (password.length < 8) {
      setError('비밀번호는 최소 8자 이상이어야 합니다')
      return
    }

    setIsLoading(true)

    try {
      await signup({ email, username, password })
      router.push('/dashboard')
    } catch (err: any) {
      setError(err.response?.data?.message || '회원가입에 실패했습니다')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950 p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-2xl">A</span>
            </div>
            <span className="text-white font-bold text-3xl">AIPX</span>
          </Link>
        </div>

        <Card className="bg-gray-900 border-gray-800">
          <CardHeader>
            <CardTitle className="text-2xl text-center">회원가입</CardTitle>
            <CardDescription className="text-center">
              AIPX 계정을 만들어 시작하세요
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Email Input */}
              <div className="space-y-2">
                <label htmlFor="email" className="text-sm font-medium text-gray-300">
                  이메일
                </label>
                <Input
                  id="email"
                  type="email"
                  placeholder="your@email.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="bg-gray-800 border-gray-700"
                  aria-label="이메일 주소"
                />
              </div>

              {/* Username Input */}
              <div className="space-y-2">
                <label htmlFor="username" className="text-sm font-medium text-gray-300">
                  사용자명
                </label>
                <Input
                  id="username"
                  type="text"
                  placeholder="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                  className="bg-gray-800 border-gray-700"
                  aria-label="사용자명"
                />
              </div>

              {/* Password Input */}
              <div className="space-y-2">
                <label htmlFor="password" className="text-sm font-medium text-gray-300">
                  비밀번호
                </label>
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={8}
                  className="bg-gray-800 border-gray-700"
                  aria-label="비밀번호"
                  aria-describedby="password-hint"
                />
                <p id="password-hint" className="text-xs text-gray-500">
                  최소 8자 이상
                </p>
              </div>

              {/* Confirm Password Input */}
              <div className="space-y-2">
                <label htmlFor="confirmPassword" className="text-sm font-medium text-gray-300">
                  비밀번호 확인
                </label>
                <Input
                  id="confirmPassword"
                  type="password"
                  placeholder="••••••••"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  className="bg-gray-800 border-gray-700"
                  aria-label="비밀번호 확인"
                />
              </div>

              {/* Error Message */}
              {error && (
                <div
                  className="bg-red-500/10 border border-red-500/50 rounded-md p-3 text-sm text-red-500"
                  role="alert"
                  aria-live="assertive"
                >
                  {error}
                </div>
              )}

              {/* Submit Button */}
              <Button
                type="submit"
                className="w-full"
                disabled={isLoading}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    계정 생성 중...
                  </>
                ) : (
                  '회원가입'
                )}
              </Button>

              {/* Login Link */}
              <p className="text-center text-sm text-gray-400">
                이미 계정이 있으신가요?{' '}
                <Link href="/login" className="text-blue-500 hover:text-blue-400">
                  로그인
                </Link>
              </p>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
