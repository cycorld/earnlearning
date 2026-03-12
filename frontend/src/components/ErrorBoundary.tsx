import { Component, type ReactNode } from 'react'

interface Props { children: ReactNode }
interface State { hasError: boolean; error?: Error }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center p-8 text-center">
          <p className="text-lg font-semibold text-destructive">오류가 발생했습니다</p>
          <p className="mt-2 text-sm text-muted-foreground">{this.state.error?.message}</p>
          <button
            className="mt-4 rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
            onClick={() => { this.setState({ hasError: false }); window.location.reload() }}
          >
            새로고침
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
