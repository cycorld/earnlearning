export interface User {
  id: number
  email: string
  name: string
  role: 'admin' | 'student'
  status: 'pending' | 'approved' | 'rejected'
  department: string
  student_id: string
  bio: string
  avatar_url: string
  wallet_balance?: number
  total_asset_value?: number
  company_count?: number
}

export interface Wallet {
  balance: number
  total_asset_value: number
  asset_breakdown: AssetBreakdown
  rank: number
  total_students: number
}

export interface AssetBreakdown {
  cash: number
  stock_value: number
  company_equity: number
  total_debt: number
}

export interface Transaction {
  id: number
  amount: number
  balance_after: number
  tx_type: string
  description: string
  reference_type: string
  reference_id: number
  created_at: string
}

export interface Company {
  id: number
  owner?: { id: number; name: string; student_id: string }
  name: string
  description: string
  logo_url: string
  initial_capital: number
  total_capital: number
  total_shares: number
  valuation: number
  listed: boolean
  wallet_balance?: number
  shareholders?: Shareholder[]
  my_shares?: number
  my_percentage?: number
  business_card?: string
  status: string
  created_at: string
}

export interface Shareholder {
  user_id: number
  name: string
  shares: number
  percentage: number
  acquisition_type: string
}

export interface Channel {
  id: number
  name: string
  slug: string
  channel_type: string
  write_role: string
}

export interface Post {
  id: number
  channel?: Channel
  author?: { id: number; name: string; avatar_url: string; student_id: string }
  content: string
  post_type: string
  media: MediaItem[]
  tags: string[]
  like_count: number
  comment_count: number
  is_liked: boolean
  pinned: boolean
  created_at: string
}

export interface MediaItem {
  url: string
  type: string
  name: string
}

export interface Comment {
  id: number
  post_id: number
  author?: { id: number; name: string; avatar_url: string }
  content: string
  media: MediaItem[]
  created_at: string
}

export interface Assignment {
  id: number
  post_id: number
  deadline: string
  reward_amount: number
  max_score: number
}

export interface Submission {
  id: number
  assignment_id: number
  student_id: number
  comment_id?: number
  content: string
  files: string
  grade?: number
  rewarded: boolean
  submitted_at: string
}

export interface FreelanceJob {
  id: number
  client?: { id: number; name: string; rating: number }
  title: string
  description: string
  budget: number
  deadline?: string
  required_skills: string[]
  status: string
  freelancer_id?: number
  escrow_amount: number
  agreed_price: number
  application_count?: number
  work_completed: boolean
  created_at: string
}

export interface JobApplication {
  id: number
  job_id: number
  user?: { id: number; name: string; rating: number }
  proposal: string
  price: number
  status: string
  created_at: string
}

export interface InvestmentRound {
  id: number
  company?: { id: number; name: string; valuation: number; logo_url: string }
  owner?: { id: number; name: string }
  target_amount: number
  offered_percent: number
  current_amount: number
  price_per_share: number
  new_shares: number
  status: string
  expires_at?: string
  created_at: string
}

export interface Investment {
  company: { id: number; name: string; valuation: number }
  shares: number
  percentage: number
  invested_amount: number
  current_value: number
  profit: number
  dividends_received: number
}

export interface Loan {
  id: number
  amount: number
  remaining: number
  interest_rate: number
  penalty_rate: number
  purpose: string
  status: string
  next_payment?: string
  weekly_interest?: number
  created_at: string
}

export interface Notification {
  id: number
  notif_type: string
  title: string
  body: string
  reference_type: string
  reference_id: number
  is_read: boolean
  created_at: string
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  error?: {
    code: string
    message: string
  }
}

export interface Pagination {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface PaginatedData<T> {
  data: T[]
  pagination: Pagination
}
