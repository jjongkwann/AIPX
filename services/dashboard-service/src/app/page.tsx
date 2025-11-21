import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { ArrowRight, Brain, TrendingUp, Shield, Zap } from 'lucide-react'

/**
 * Landing page component
 *
 * Features:
 * - Hero section
 * - Feature highlights
 * - CTA buttons
 * - Responsive design
 *
 * Accessibility:
 * - Semantic HTML
 * - ARIA landmarks
 * - Keyboard navigation
 */
export default function LandingPage() {
  return (
    <main className="min-h-screen bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950">
      {/* Navigation */}
      <nav className="border-b border-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-xl">A</span>
              </div>
              <span className="text-white font-bold text-2xl">AIPX</span>
            </div>

            <div className="flex gap-4">
              <Link href="/login">
                <Button variant="ghost" className="text-gray-300 hover:text-white">
                  로그인
                </Button>
              </Link>
              <Link href="/signup">
                <Button>시작하기</Button>
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center space-y-8">
          <h1 className="text-5xl md:text-7xl font-bold text-white">
            AI 기반 투자 플랫폼
            <span className="block text-blue-500 mt-2">AIPX</span>
          </h1>

          <p className="text-xl md:text-2xl text-gray-400 max-w-3xl mx-auto">
            인공지능이 분석하고, 실행하고, 최적화하는 <br />
            차세대 자동 투자 시스템
          </p>

          <div className="flex flex-col sm:flex-row gap-4 justify-center pt-8">
            <Link href="/signup">
              <Button size="lg" className="gap-2 text-lg px-8 py-6">
                무료로 시작하기
                <ArrowRight className="w-5 h-5" />
              </Button>
            </Link>
            <Link href="/dashboard">
              <Button size="lg" variant="outline" className="text-lg px-8 py-6">
                데모 둘러보기
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24">
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
          <FeatureCard
            icon={<Brain className="w-12 h-12 text-blue-500" />}
            title="AI 에이전트"
            description="GPT-4 기반 인지 엔진이 시장을 분석하고 최적의 투자 전략을 제안합니다"
          />
          <FeatureCard
            icon={<TrendingUp className="w-12 h-12 text-green-500" />}
            title="자동 실행"
            description="전략이 수립되면 자동으로 매매를 실행하고 포트폴리오를 관리합니다"
          />
          <FeatureCard
            icon={<Shield className="w-12 h-12 text-purple-500" />}
            title="리스크 관리"
            description="실시간 리스크 모니터링과 자동 손절로 자산을 보호합니다"
          />
          <FeatureCard
            icon={<Zap className="w-12 h-12 text-yellow-500" />}
            title="빠른 백테스트"
            description="전략을 빠르게 검증하고 과거 데이터로 성과를 시뮬레이션합니다"
          />
        </div>
      </section>

      {/* Stats Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 border-t border-gray-800">
        <div className="grid md:grid-cols-3 gap-8 text-center">
          <div>
            <p className="text-5xl font-bold text-white mb-2">99.9%</p>
            <p className="text-gray-400">시스템 가동률</p>
          </div>
          <div>
            <p className="text-5xl font-bold text-white mb-2">&lt;100ms</p>
            <p className="text-gray-400">평균 응답 시간</p>
          </div>
          <div>
            <p className="text-5xl font-bold text-white mb-2">24/7</p>
            <p className="text-gray-400">무중단 모니터링</p>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24">
        <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-2xl p-12 text-center">
          <h2 className="text-4xl font-bold text-white mb-4">
            지금 시작하세요
          </h2>
          <p className="text-xl text-gray-100 mb-8">
            AI 투자의 새로운 시대를 경험해보세요
          </p>
          <Link href="/signup">
            <Button size="lg" variant="secondary" className="text-lg px-8 py-6">
              무료 계정 만들기
            </Button>
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center text-gray-500">
          <p>&copy; 2025 AIPX. All rights reserved.</p>
        </div>
      </footer>
    </main>
  )
}

/**
 * Feature card component
 */
function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode
  title: string
  description: string
}) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-gray-700 transition-colors">
      <div className="mb-4">{icon}</div>
      <h3 className="text-xl font-semibold text-white mb-2">{title}</h3>
      <p className="text-gray-400">{description}</p>
    </div>
  )
}
