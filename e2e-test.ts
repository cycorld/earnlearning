import { chromium, Browser, Page } from 'playwright';
import * as path from 'path';

const BASE = 'http://localhost:5173';
const API = 'http://localhost:8090/api';
const SCREENSHOT_DIR = path.resolve(__dirname, 'screenshots');

let browser: Browser;
let page: Page;

async function screenshot(name: string) {
  await page.screenshot({ path: path.join(SCREENSHOT_DIR, `${name}.png`), fullPage: true });
  console.log(`  📸 ${name}.png`);
}

async function sleep(ms: number) {
  await new Promise(r => setTimeout(r, ms));
}

async function main() {
  browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 }, // iPhone 14 size
    locale: 'ko-KR',
  });
  page = await context.newPage();

  let adminToken = '';
  let studentToken = '';
  let classroomCode = '';

  try {
    // ============================================================
    // 1. 로그인 페이지
    // ============================================================
    console.log('\n🧪 시나리오 1: 로그인 페이지');
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 10000 });
    await screenshot('01-login-page');

    // ============================================================
    // 2. 회원가입 페이지
    // ============================================================
    console.log('\n🧪 시나리오 2: 회원가입 페이지');
    await page.click('a[href="/register"]');
    await page.waitForSelector('input[id="name"]', { timeout: 5000 });
    await screenshot('02-register-page-empty');

    // Fill registration form
    await page.fill('input[id="name"]', '김이화');
    await page.fill('input[id="email"]', 'student@ewha.ac.kr');
    await page.fill('input[id="password"]', 'password1234');
    await page.fill('input[id="department"]', '컴퓨터공학과');
    await page.fill('input[id="student_id_display"], input[id="student_id"], input[id="studentId"]', '2024001');
    await screenshot('02-register-page-filled');

    // Submit registration
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('02-register-result');

    // ============================================================
    // 3. 관리자 로그인
    // ============================================================
    console.log('\n🧪 시나리오 3: 관리자 로그인');
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'admin@ewha.ac.kr');
    await page.fill('input[type="password"]', 'admin1234');
    await screenshot('03-admin-login-filled');

    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('03-admin-after-login');

    // Get admin token from API for direct calls
    const adminResp = await fetch(`${API}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'admin@ewha.ac.kr', password: 'admin1234' }),
    });
    const adminData = await adminResp.json();
    adminToken = adminData.data?.token || '';

    // ============================================================
    // 4. 관리자 대시보드
    // ============================================================
    console.log('\n🧪 시나리오 4: 관리자 대시보드');
    await page.goto(BASE + '/admin');
    await sleep(1500);
    await screenshot('04-admin-dashboard');

    // ============================================================
    // 5. 사용자 승인 페이지
    // ============================================================
    console.log('\n🧪 시나리오 5: 사용자 관리 (승인)');
    await page.goto(BASE + '/admin/users');
    await sleep(1500);
    await screenshot('05-admin-users-pending');

    // Approve via API
    if (adminToken) {
      const pendingResp = await fetch(`${API}/admin/users/pending`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      const pendingData = await pendingResp.json();
      if (pendingData.data?.length > 0) {
        const studentId = pendingData.data[0].id;
        await fetch(`${API}/admin/users/${studentId}/approve`, {
          method: 'PUT',
          headers: { Authorization: `Bearer ${adminToken}` },
        });
        console.log(`  ✅ 학생 ${studentId} 승인 완료`);
      }
    }

    // ============================================================
    // 6. 클래스룸 생성
    // ============================================================
    console.log('\n🧪 시나리오 6: 클래스룸 생성');
    await page.goto(BASE + '/admin/classroom');
    await sleep(1500);
    await screenshot('06-admin-classroom');

    // Create classroom via API
    if (adminToken) {
      const classResp = await fetch(`${API}/classrooms`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${adminToken}` },
        body: JSON.stringify({ name: '2026 스타트업 코딩입문', initial_capital: 50000000 }),
      });
      const classData = await classResp.json();
      classroomCode = classData.data?.code || '';
      console.log(`  📋 클래스룸 코드: ${classroomCode}`);

      await page.reload();
      await sleep(1500);
      await screenshot('06-admin-classroom-created');
    }

    // ============================================================
    // 7. 대출 관리 페이지
    // ============================================================
    console.log('\n🧪 시나리오 7: 대출 관리');
    await page.goto(BASE + '/admin/loans');
    await sleep(1500);
    await screenshot('07-admin-loans');

    // ============================================================
    // 8. KPI 관리 페이지
    // ============================================================
    console.log('\n🧪 시나리오 8: KPI 관리');
    await page.goto(BASE + '/admin/kpi');
    await sleep(1500);
    await screenshot('08-admin-kpi');

    // ============================================================
    // 9. 학생 로그인
    // ============================================================
    console.log('\n🧪 시나리오 9: 학생 로그인');
    // Logout first
    await page.goto(BASE + '/profile');
    await sleep(1000);
    const logoutBtn = page.locator('button:has-text("로그아웃")');
    if (await logoutBtn.isVisible()) {
      await logoutBtn.click();
      await sleep(1000);
    }

    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'student@ewha.ac.kr');
    await page.fill('input[type="password"]', 'password1234');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('09-student-login-result');

    // Get student token
    const stuResp = await fetch(`${API}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'student@ewha.ac.kr', password: 'password1234' }),
    });
    const stuData = await stuResp.json();
    studentToken = stuData.data?.token || '';

    // ============================================================
    // 10. 피드 페이지 (클래스룸 참여 전)
    // ============================================================
    console.log('\n🧪 시나리오 10: 피드 (클래스룸 참여 전)');
    await page.goto(BASE + '/feed');
    await sleep(1500);
    await screenshot('10-feed-no-classroom');

    // Join classroom via API
    if (studentToken && classroomCode) {
      await fetch(`${API}/classrooms/join`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${studentToken}` },
        body: JSON.stringify({ code: classroomCode }),
      });
      console.log(`  ✅ 클래스룸 참여 완료`);
    }

    // ============================================================
    // 11. 피드 페이지 (클래스룸 참여 후)
    // ============================================================
    console.log('\n🧪 시나리오 11: 피드 (클래스룸 참여 후)');
    await page.goto(BASE + '/feed');
    await sleep(2000);
    await screenshot('11-feed-with-classroom');

    // ============================================================
    // 12. 지갑 페이지
    // ============================================================
    console.log('\n🧪 시나리오 12: 지갑');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('12-wallet');

    // ============================================================
    // 13. 거래내역
    // ============================================================
    console.log('\n🧪 시나리오 13: 거래내역');
    await page.goto(BASE + '/wallet/transactions');
    await sleep(1500);
    await screenshot('13-transactions');

    // ============================================================
    // 14. 회사 목록 (비어있음)
    // ============================================================
    console.log('\n🧪 시나리오 14: 회사 목록');
    await page.goto(BASE + '/company');
    await sleep(1500);
    await screenshot('14-company-list-empty');

    // ============================================================
    // 15. 회사 설립
    // ============================================================
    console.log('\n🧪 시나리오 15: 회사 설립');
    await page.goto(BASE + '/company/new');
    await sleep(1500);
    await screenshot('15-company-new-empty');

    // Create company via API
    if (studentToken) {
      const compResp = await fetch(`${API}/companies`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${studentToken}` },
        body: JSON.stringify({ name: '이화AI랩', description: 'AI 기반 교육 스타트업', initial_capital: 5000000 }),
      });
      const compData = await compResp.json();
      console.log(`  ✅ 회사 설립: ${compData.data?.name || 'error'}`);
    }

    // ============================================================
    // 16. 회사 목록 (설립 후)
    // ============================================================
    console.log('\n🧪 시나리오 16: 회사 목록 (설립 후)');
    await page.goto(BASE + '/company');
    await sleep(1500);
    await screenshot('16-company-list-with-company');

    // ============================================================
    // 17. 회사 상세
    // ============================================================
    console.log('\n🧪 시나리오 17: 회사 상세');
    await page.goto(BASE + '/company/1');
    await sleep(1500);
    await screenshot('17-company-detail');

    // ============================================================
    // 18. 지갑 (회사설립 후)
    // ============================================================
    console.log('\n🧪 시나리오 18: 지갑 (회사설립 후)');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('18-wallet-after-company');

    // ============================================================
    // 19. 프리랜서 마켓
    // ============================================================
    console.log('\n🧪 시나리오 19: 프리랜서 마켓');
    await page.goto(BASE + '/market');
    await sleep(1500);
    await screenshot('19-market-empty');

    // Create a freelance job via API
    if (studentToken) {
      await fetch(`${API}/freelance/jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${studentToken}` },
        body: JSON.stringify({
          title: '랜딩페이지 제작',
          description: '이화AI랩 소개 랜딩페이지 제작',
          budget: 500000,
          deadline: '2026-04-01',
          required_skills: 'React,TypeScript,Tailwind',
        }),
      });
    }

    await page.goto(BASE + '/market');
    await sleep(1500);
    await screenshot('19-market-with-job');

    // ============================================================
    // 20. 프리랜서 마켓 새 의뢰 작성
    // ============================================================
    console.log('\n🧪 시나리오 20: 프리랜서 의뢰 작성');
    await page.goto(BASE + '/market/new');
    await sleep(1500);
    await screenshot('20-market-new');

    // ============================================================
    // 21. 프리랜서 마켓 상세
    // ============================================================
    console.log('\n🧪 시나리오 21: 프리랜서 의뢰 상세');
    await page.goto(BASE + '/market/1');
    await sleep(1500);
    await screenshot('21-market-detail');

    // ============================================================
    // 22. 투자 페이지
    // ============================================================
    console.log('\n🧪 시나리오 22: 투자');
    await page.goto(BASE + '/invest');
    await sleep(1500);
    await screenshot('22-invest');

    // ============================================================
    // 23. 거래소 페이지
    // ============================================================
    console.log('\n🧪 시나리오 23: 거래소');
    await page.goto(BASE + '/exchange');
    await sleep(1500);
    await screenshot('23-exchange');

    // ============================================================
    // 24. 은행 페이지
    // ============================================================
    console.log('\n🧪 시나리오 24: 은행');
    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('24-bank-empty');

    // ============================================================
    // 25. 대출 신청
    // ============================================================
    console.log('\n🧪 시나리오 25: 대출 신청');
    await page.goto(BASE + '/bank/apply');
    await sleep(1500);
    await screenshot('25-loan-apply');

    // Apply loan via API
    if (studentToken) {
      await fetch(`${API}/loans`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${studentToken}` },
        body: JSON.stringify({ amount: 2000000, purpose: '마케팅 비용' }),
      });
    }

    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('25-bank-with-loan');

    // Approve loan via admin
    if (adminToken) {
      const loansResp = await fetch(`${API}/admin/loans`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      const loansData = await loansResp.json();
      const loans = loansData.data?.loans || loansData.data || [];
      const pendingLoan = Array.isArray(loans) ? loans.find((l: any) => l.status === 'pending') : null;
      if (pendingLoan) {
        await fetch(`${API}/admin/loans/${pendingLoan.id}/approve`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${adminToken}` },
          body: JSON.stringify({ interest_rate: 5.0 }),
        });
        console.log(`  ✅ 대출 승인 완료`);
      }
    }

    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('25-bank-loan-approved');

    // ============================================================
    // 26. 프로필 페이지
    // ============================================================
    console.log('\n🧪 시나리오 26: 프로필');
    await page.goto(BASE + '/profile');
    await sleep(1500);
    await screenshot('26-profile');

    // ============================================================
    // 27. 알림 페이지
    // ============================================================
    console.log('\n🧪 시나리오 27: 알림');
    await page.goto(BASE + '/notifications');
    await sleep(1500);
    await screenshot('27-notifications');

    // ============================================================
    // 28. 지갑 최종 상태 (모든 활동 후)
    // ============================================================
    console.log('\n🧪 시나리오 28: 지갑 최종 상태');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('28-wallet-final');

    await page.goto(BASE + '/wallet/transactions');
    await sleep(1500);
    await screenshot('28-transactions-final');

    // ============================================================
    // 29. Bottom Nav 더보기 메뉴
    // ============================================================
    console.log('\n🧪 시나리오 29: 하단 네비게이션 더보기');
    await page.goto(BASE + '/feed');
    await sleep(1000);
    const moreBtn = page.locator('button:has-text("더보기"), nav button:last-child');
    if (await moreBtn.first().isVisible()) {
      await moreBtn.first().click();
      await sleep(800);
      await screenshot('29-bottom-nav-more');
    }

    // ============================================================
    // 30. 헤더 알림 벨
    // ============================================================
    console.log('\n🧪 시나리오 30: 헤더');
    await page.goto(BASE + '/feed');
    await sleep(1000);
    await screenshot('30-header');

    console.log('\n✅ 모든 시나리오 테스트 완료!');
    console.log(`📁 스크린샷 저장 위치: ${SCREENSHOT_DIR}`);

  } catch (err) {
    console.error('\n❌ 테스트 실패:', err);
    await screenshot('error-state');
  } finally {
    await browser.close();
  }
}

main();
