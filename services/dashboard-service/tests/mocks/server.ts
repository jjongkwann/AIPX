import { setupServer } from 'msw/node'
import { handlers } from './handlers'

// Setup requests interception using the given handlers
export const server = setupServer(...handlers)

// Start server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))

// Reset handlers after each test
afterEach(() => server.resetHandlers())

// Clean up after all tests are done
afterAll(() => server.close())
