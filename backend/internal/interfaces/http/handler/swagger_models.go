package handler

// APIResponse is the standard API response envelope.
type APIResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data"`
	Error   *APIError   `json:"error"`
}

// APIError represents an error in the API response.
type APIError struct {
	Code    string `json:"code" example:"INVALID_INPUT"`
	Message string `json:"message" example:"잘못된 입력입니다"`
}

// --- Auth models ---

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email     string `json:"email" example:"student@ewha.ac.kr"`
	Password  string `json:"password" example:"password123"`
	Name      string `json:"name" example:"홍길동"`
	StudentID string `json:"student_id" example:"2024001"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" example:"student@ewha.ac.kr"`
	Password string `json:"password" example:"password123"`
}

// TokenResponse is the JWT token response.
type TokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// --- Wallet models ---

// TransferRequest represents a wallet transfer request.
type TransferRequest struct {
	TargetUserID int    `json:"target_user_id" example:"2"`
	Amount       int    `json:"amount" example:"1000"`
	Description  string `json:"description" example:"송금 메모"`
}

// AdminTransferRequest represents an admin batch transfer.
type AdminTransferRequest struct {
	UserIDs     []int  `json:"user_ids"`
	Amount      int    `json:"amount" example:"5000"`
	Description string `json:"description" example:"보상금 지급"`
}

// --- Classroom models ---

// CreateClassroomRequest represents a classroom creation request.
type CreateClassroomRequest struct {
	Name string `json:"name" example:"코딩입문 1반"`
}

// JoinClassroomRequest represents a classroom join request.
type JoinClassroomRequest struct {
	Code string `json:"code" example:"ABC123"`
}

// --- Company models ---

// CreateCompanyRequest represents a company creation request.
type CreateCompanyRequest struct {
	Name        string `json:"name" example:"테크스타트업"`
	Description string `json:"description" example:"AI 기반 서비스"`
	Capital     int    `json:"capital" example:"100000"`
}

// UpdateCompanyRequest represents a company update request.
type UpdateCompanyRequest struct {
	Description string `json:"description" example:"수정된 설명"`
	LogoURL     string `json:"logo_url" example:"/uploads/logo.png"`
}

// --- Post models ---

// CreatePostRequest represents a post creation request.
type CreatePostRequest struct {
	ChannelID int      `json:"channel_id" example:"1"`
	Content   string   `json:"content" example:"게시글 내용"`
	Media     []string `json:"media"`
	Tags      []string `json:"tags"`
}

// UpdatePostRequest represents a post update request.
type UpdatePostRequest struct {
	Content string   `json:"content" example:"수정된 내용"`
	Media   []string `json:"media"`
	Tags    []string `json:"tags"`
}

// CreateCommentRequest represents a comment creation request.
type CreateCommentRequest struct {
	PostID  int    `json:"post_id" example:"1"`
	Content string `json:"content" example:"댓글 내용"`
}

// CreateAssignmentRequest represents an assignment creation request.
type CreateAssignmentRequest struct {
	ChannelID   int    `json:"channel_id" example:"1"`
	Title       string `json:"title" example:"과제 제목"`
	Description string `json:"description" example:"과제 설명"`
	DueDate     string `json:"due_date" example:"2026-04-01"`
	MaxScore    int    `json:"max_score" example:"100"`
}

// SubmitAssignmentRequest represents an assignment submission.
type SubmitAssignmentRequest struct {
	AssignmentID int    `json:"assignment_id" example:"1"`
	Content      string `json:"content" example:"제출 내용"`
	FileURL      string `json:"file_url" example:"/uploads/file.pdf"`
}

// GradeAssignmentRequest represents a grading request.
type GradeAssignmentRequest struct {
	SubmissionID int    `json:"submission_id" example:"1"`
	Score        int    `json:"score" example:"95"`
	Feedback     string `json:"feedback" example:"잘했습니다"`
}

// --- Freelance models ---

// CreateJobRequest represents a freelance job creation request.
type CreateJobRequest struct {
	Title       string   `json:"title" example:"웹사이트 제작"`
	Description string   `json:"description" example:"React 기반 랜딩페이지"`
	Budget      int      `json:"budget" example:"50000"`
	Skills      []string `json:"skills"`
}

// ApplyJobRequest represents a job application.
type ApplyJobRequest struct {
	Proposal string `json:"proposal" example:"지원 내용"`
	Price    int    `json:"price" example:"45000"`
}

// CompleteWorkRequest represents work completion.
type CompleteWorkRequest struct {
	Report string   `json:"report" example:"완료 보고서"`
	Media  []string `json:"media"`
}

// ReviewJobRequest represents a job review.
type ReviewJobRequest struct {
	Rating  int    `json:"rating" example:"5"`
	Comment string `json:"comment" example:"좋은 작업이었습니다"`
}

// --- Grant models ---

// CreateGrantRequest represents a grant creation request.
type CreateGrantRequest struct {
	Title         string `json:"title" example:"창업지원금"`
	Description   string `json:"description" example:"지원 설명"`
	Reward        int    `json:"reward" example:"100000"`
	MaxApplicants int    `json:"max_applicants" example:"5"`
}

// ApplyGrantRequest represents a grant application.
type ApplyGrantRequest struct {
	Proposal string `json:"proposal" example:"지원 제안서"`
}

// --- Investment models ---

// CreateRoundRequest represents an investment round creation.
type CreateRoundRequest struct {
	CompanyID  int `json:"company_id" example:"1"`
	PricePerShare int `json:"price_per_share" example:"1000"`
	TotalShares   int `json:"total_shares" example:"100"`
}

// ExecuteDividendRequest represents a dividend execution.
type ExecuteDividendRequest struct {
	CompanyID     int `json:"company_id" example:"1"`
	AmountPerShare int `json:"amount_per_share" example:"100"`
}

// CreateKpiRuleRequest represents a KPI rule creation.
type CreateKpiRuleRequest struct {
	CompanyID int    `json:"company_id" example:"1"`
	Name      string `json:"name" example:"매출 목표"`
	Target    int    `json:"target" example:"1000000"`
}

// AddKpiRevenueRequest represents adding KPI revenue.
type AddKpiRevenueRequest struct {
	RuleID int `json:"rule_id" example:"1"`
	Amount int `json:"amount" example:"50000"`
}

// --- Exchange models ---

// PlaceOrderRequest represents a stock order.
type PlaceOrderRequest struct {
	CompanyID int    `json:"company_id" example:"1"`
	Side      string `json:"side" example:"buy"`
	Price     int    `json:"price" example:"1000"`
	Quantity  int    `json:"quantity" example:"10"`
}

// --- Loan models ---

// ApplyLoanRequest represents a loan application.
type ApplyLoanRequest struct {
	Amount  int    `json:"amount" example:"50000"`
	Purpose string `json:"purpose" example:"사업 자금"`
}

// ApproveLoanRequest represents loan approval.
type ApproveLoanRequest struct {
	InterestRate float64 `json:"interest_rate" example:"5.0"`
	Weeks        int     `json:"weeks" example:"4"`
}

// RepayLoanRequest represents a loan repayment.
type RepayLoanRequest struct {
	Amount int `json:"amount" example:"10000"`
}

// --- Notification models ---

// SubscribePushRequest represents a push subscription.
type SubscribePushRequest struct {
	Endpoint string `json:"endpoint" example:"https://fcm.googleapis.com/..."`
	P256dh   string `json:"p256dh" example:"BNcRd..."`
	Auth     string `json:"auth" example:"tBHI..."`
}

// UnsubscribePushRequest represents a push unsubscription.
type UnsubscribePushRequest struct {
	Endpoint string `json:"endpoint" example:"https://fcm.googleapis.com/..."`
}

// UpdateEmailPrefRequest represents email preference update.
type UpdateEmailPrefRequest struct {
	EmailEnabled bool `json:"email_enabled" example:"true"`
}

// AnnouncementRequest represents an admin announcement.
type AnnouncementRequest struct {
	Title      string `json:"title" example:"공지사항 제목"`
	Body       string `json:"body" example:"공지사항 내용"`
	UserIDs    []int  `json:"user_ids"`
	SendNotify *bool  `json:"send_notify" example:"true"`
}

// MessageResponse is a simple message response.
type MessageResponse struct {
	Message string `json:"message" example:"성공적으로 처리되었습니다"`
}

// AcceptApplicationRequest represents accepting a job application.
type AcceptApplicationRequest struct {
	ApplicationID int `json:"application_id" example:"1"`
}

// --- OAuth response models (typed wrappers, RFC 6749 §5.1 호환) ---

// OAuthTokenData is the data payload of /oauth/token (success).
type OAuthTokenData struct {
	AccessToken  string   `json:"access_token" example:"a1b2c3..."`
	RefreshToken string   `json:"refresh_token" example:"d4e5f6..."`
	TokenType    string   `json:"token_type" example:"Bearer"`
	ExpiresIn    int      `json:"expires_in" example:"3600"`
	Scopes       []string `json:"scopes" example:"read:profile,read:wallet"`
}

// OAuthTokenResponse — POST /oauth/token success envelope.
type OAuthTokenResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    *OAuthTokenData `json:"data"`
	Error   *APIError       `json:"error"`
}

// OAuthUserInfoData is the data payload of /oauth/userinfo.
type OAuthUserInfoData struct {
	ID         int    `json:"id" example:"1"`
	Email      string `json:"email" example:"student@ewha.ac.kr"`
	Name       string `json:"name" example:"홍길동"`
	Department string `json:"department" example:"행정학과"`
	Bio        string `json:"bio"`
	AvatarURL  string `json:"avatar_url" example:"/uploads/avatar.png"`
}

// OAuthUserInfoResponse — GET /oauth/userinfo success envelope.
type OAuthUserInfoResponse struct {
	Success bool               `json:"success" example:"true"`
	Data    *OAuthUserInfoData `json:"data"`
	Error   *APIError          `json:"error"`
}

// OAuthTokenRequest documents POST /oauth/token body.
// PKCE 퍼블릭 클라이언트는 client_secret 없이 code_verifier 만으로도 가능.
// refresh_token grant 또한 PKCE 발급 토큰은 client_secret 옵셔널.
type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type" example:"authorization_code" enums:"authorization_code,refresh_token"`
	Code         string `json:"code,omitempty" example:"a1b2c3..."`
	ClientID     string `json:"client_id" example:"abcd1234"`
	ClientSecret string `json:"client_secret,omitempty" example:"(PKCE 사용 시 생략 가능)"`
	RedirectURI  string `json:"redirect_uri,omitempty" example:"https://myapp.com/callback"`
	CodeVerifier string `json:"code_verifier,omitempty" example:"(PKCE 사용 시 필수)"`
	RefreshToken string `json:"refresh_token,omitempty" example:"d4e5f6..."`
}

// BusinessCardRequest represents a business card creation request.
type BusinessCardRequest struct {
	Name     string `json:"name" example:"홍길동"`
	Title    string `json:"title" example:"CEO"`
	Email    string `json:"email" example:"ceo@company.com"`
	Phone    string `json:"phone" example:"010-1234-5678"`
	Template string `json:"template" example:"modern"`
}
