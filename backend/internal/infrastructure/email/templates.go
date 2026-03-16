package email

import (
	"fmt"
	"strings"
)

func FormatNotificationEmail(title, body, refType string, refID int) (subject, html, text string) {
	subject = fmt.Sprintf("[EarnLearning] %s", title)

	linkURL := ""
	if refType != "" && refID > 0 {
		linkURL = fmt.Sprintf("https://earnlearning.com/%s/%d", refType, refID)
	}

	var linkHTML string
	if linkURL != "" {
		linkHTML = fmt.Sprintf(`<p style="margin-top:20px;"><a href="%s" style="background-color:#6366f1;color:#fff;padding:10px 20px;border-radius:6px;text-decoration:none;font-weight:600;">자세히 보기</a></p>`, linkURL)
	}

	html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background-color:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<div style="max-width:480px;margin:20px auto;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);">
  <div style="background:linear-gradient(135deg,#6366f1,#8b5cf6);padding:24px;text-align:center;">
    <h1 style="color:#fff;margin:0;font-size:20px;">EarnLearning</h1>
  </div>
  <div style="padding:24px;">
    <h2 style="margin:0 0 12px;font-size:18px;color:#1e293b;">%s</h2>
    <p style="margin:0;font-size:15px;color:#475569;line-height:1.6;">%s</p>
    %s
  </div>
  <div style="padding:16px 24px;background:#f8fafc;text-align:center;font-size:12px;color:#94a3b8;">
    <p style="margin:0;">이화여자대학교 스타트업을 위한 코딩입문</p>
    <p style="margin:4px 0 0;"><a href="https://earnlearning.com/profile" style="color:#6366f1;text-decoration:none;">이메일 알림 설정 변경</a></p>
  </div>
</div>
</body>
</html>`, title, strings.ReplaceAll(body, "\n", "<br>"), linkHTML)

	text = fmt.Sprintf("[EarnLearning] %s\n\n%s", title, body)
	if linkURL != "" {
		text += fmt.Sprintf("\n\n자세히 보기: %s", linkURL)
	}

	return subject, html, text
}
