// #166 학생별 이메일 수신함 — 수신 메일을 파싱해 LMS 백엔드 웹훅으로 전달.
// 흐름: Email Routing catch-all → email() → postal-mime 파싱 → POST /api/mail/inbound
// - 백엔드 404(없는 주소) → setReject 로 영구 반송
// - 백엔드 5xx/네트워크 오류 → throw 로 일시 오류 (SMTP 재시도 유도)
import PostalMime from "postal-mime";

interface Env {
  BACKEND_INBOUND_URL: string;
  MAIL_WEBHOOK_SECRET: string;
}

// 첨부 총량 상한 (백엔드 413 상한과 동일하게 유지)
const MAX_ATTACHMENT_TOTAL = 10 * 1024 * 1024;

function toBase64(data: Uint8Array): string {
  let binary = "";
  const chunk = 0x8000;
  for (let i = 0; i < data.length; i += chunk) {
    binary += String.fromCharCode(...data.subarray(i, i + chunk));
  }
  return btoa(binary);
}

export default {
  async email(message: ForwardableEmailMessage, env: Env, _ctx: ExecutionContext): Promise<void> {
    // raw 스트림은 1회용 — 반드시 먼저 버퍼링
    const raw = await new Response(message.raw).arrayBuffer();
    const parsed = await PostalMime.parse(raw);

    let attachmentTotal = 0;
    const attachments: { filename: string; mime: string; content_base64: string }[] = [];
    for (const att of parsed.attachments ?? []) {
      const content =
        typeof att.content === "string"
          ? new TextEncoder().encode(att.content)
          : new Uint8Array(att.content);
      attachmentTotal += content.length;
      if (attachmentTotal > MAX_ATTACHMENT_TOTAL) {
        // 상한 초과분은 제외하고 본문만 전달 (메일 자체를 반송하지 않음)
        break;
      }
      attachments.push({
        filename: att.filename || "attachment",
        mime: att.mimeType || "application/octet-stream",
        content_base64: toBase64(content),
      });
    }

    const payload = {
      // 봉투 주소 사용 — 헤더 From 은 스푸핑 가능하므로 message.from 이 기준
      from: message.from,
      to: message.to,
      subject: parsed.subject ?? "",
      text: parsed.text ?? "",
      html: parsed.html ?? "",
      message_id: message.headers.get("message-id") ?? "",
      in_reply_to: message.headers.get("in-reply-to") ?? "",
      references: message.headers.get("references") ?? "",
      attachments,
    };

    const res = await fetch(env.BACKEND_INBOUND_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Mail-Webhook-Secret": env.MAIL_WEBHOOK_SECRET,
      },
      body: JSON.stringify(payload),
    });

    if (res.status === 404) {
      message.setReject("No such mailbox");
      return;
    }
    if (res.status === 413) {
      message.setReject("Message too large");
      return;
    }
    if (!res.ok) {
      // 일시 오류로 처리 → 발신 서버가 재시도
      throw new Error(`backend inbound webhook failed: ${res.status}`);
    }
  },
} satisfies ExportedHandler<Env>;
