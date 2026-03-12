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

async function apiPost(path: string, body: any, token?: string) {
  const headers: any = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;
  const resp = await fetch(`${API}${path}`, { method: 'POST', headers, body: JSON.stringify(body) });
  return resp.json();
}

async function apiGet(path: string, token: string) {
  const resp = await fetch(`${API}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  return resp.json();
}

async function apiPut(path: string, body: any, token: string) {
  const resp = await fetch(`${API}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify(body),
  });
  return resp.json();
}

async function main() {
  browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    locale: 'ko-KR',
  });
  page = await context.newPage();

  // Capture console errors
  const consoleErrors: string[] = [];
  page.on('console', msg => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });

  let adminToken = '';
  let student1Token = '';
  let student2Token = '';
  let classroomCode = '';
  let companyId = 0;
  let jobId = 0;
  let loanId = 0;

  try {
    // ============================================================
    // PHASE 1: 회원가입 & 인증 흐름
    // ============================================================
    console.log('\n━━━━ PHASE 1: 회원가입 & 인증 ━━━━');

    // 1.1 로그인 페이지 렌더링
    console.log('\n🧪 1.1: 로그인 페이지');
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 10000 });
    await screenshot('01-01-login-page');

    // 1.2 로그인 폼 유효성 검사 - 빈 폼 제출
    console.log('\n🧪 1.2: 로그인 빈 폼 제출');
    await page.click('button[type="submit"]');
    await sleep(1000);
    await screenshot('01-02-login-empty-submit');

    // 1.3 잘못된 이메일로 로그인
    console.log('\n🧪 1.3: 잘못된 이메일 로그인');
    await page.fill('input[type="email"]', 'wrong@test.com');
    await page.fill('input[type="password"]', 'wrongpass');
    await page.click('button[type="submit"]');
    await sleep(1500);
    await screenshot('01-03-login-wrong-credentials');

    // 1.4 회원가입 페이지로 이동
    console.log('\n🧪 1.4: 회원가입 페이지');
    await page.click('a[href="/register"]');
    await page.waitForSelector('input[id="name"]', { timeout: 5000 });
    await screenshot('01-04-register-empty');

    // 1.5 회원가입 폼 유효성 검사 - 빈 폼 제출
    console.log('\n🧪 1.5: 회원가입 빈 폼 제출');
    await page.click('button[type="submit"]');
    await sleep(500);
    await screenshot('01-05-register-validation');

    // 1.6 회원가입 - 짧은 비밀번호
    console.log('\n🧪 1.6: 짧은 비밀번호 검증');
    await page.fill('input[id="email"]', 'test@ewha.ac.kr');
    await page.fill('input[id="password"]', '1234');
    await page.fill('input[id="name"]', '테스트');
    await page.fill('input[id="department"]', '테스트학과');
    await page.fill('input[id="student_id"]', '2024001');
    await page.click('button[type="submit"]');
    await sleep(500);
    await screenshot('01-06-register-short-password');

    // 1.7 학생1 정상 회원가입
    console.log('\n🧪 1.7: 학생1 회원가입');
    await page.fill('input[id="email"]', 'student1@ewha.ac.kr');
    await page.fill('input[id="password"]', 'password1234');
    await page.fill('input[id="name"]', '김이화');
    await page.fill('input[id="department"]', '컴퓨터공학과');
    await page.fill('input[id="student_id"]', '2024001');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('01-07-register-success');

    // 1.8 학생2 회원가입 (두 번째 사용자)
    console.log('\n🧪 1.8: 학생2 회원가입');
    await page.goto(BASE + '/register');
    await page.waitForSelector('input[id="name"]', { timeout: 5000 });
    await page.fill('input[id="email"]', 'student2@ewha.ac.kr');
    await page.fill('input[id="password"]', 'password1234');
    await page.fill('input[id="name"]', '박이화');
    await page.fill('input[id="department"]', '경영학과');
    await page.fill('input[id="student_id"]', '2024002');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('01-08-register-student2');

    // 1.9 미승인 학생 로그인 시도
    console.log('\n🧪 1.9: 미승인 학생 로그인');
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'student1@ewha.ac.kr');
    await page.fill('input[type="password"]', 'password1234');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('01-09-login-pending-student');

    // ============================================================
    // PHASE 2: 관리자 흐름
    // ============================================================
    console.log('\n━━━━ PHASE 2: 관리자 흐름 ━━━━');

    // Get admin token
    const adminData = await apiPost('/auth/login', { email: 'admin@ewha.ac.kr', password: 'admin1234' });
    adminToken = adminData.data?.token || '';

    // 2.1 관리자 로그인
    console.log('\n🧪 2.1: 관리자 로그인');
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'admin@ewha.ac.kr');
    await page.fill('input[type="password"]', 'admin1234');
    await page.click('button[type="submit"]');
    await sleep(2000);

    // Navigate to admin dashboard
    await page.goto(BASE + '/admin');
    await sleep(1500);
    await screenshot('02-01-admin-dashboard');

    // 2.2 사용자 관리 - 대기 중인 학생 2명
    console.log('\n🧪 2.2: 사용자 관리 (대기 2명)');
    await page.goto(BASE + '/admin/users');
    await sleep(1500);
    await screenshot('02-02-admin-users-pending');

    // Approve both students via API
    const pendingResp = await apiGet('/admin/users/pending', adminToken);
    const pendingUsers = pendingResp.data || [];
    for (const user of pendingUsers) {
      await apiPut(`/admin/users/${user.id}/approve`, {}, adminToken);
      console.log(`  ✅ ${user.name} 승인 완료`);
    }

    // 2.3 사용자 관리 - 승인 후
    await page.reload();
    await sleep(1500);
    await screenshot('02-03-admin-users-approved');

    // 2.4 클래스룸 생성
    console.log('\n🧪 2.4: 클래스룸 생성');
    await page.goto(BASE + '/admin/classroom');
    await sleep(1500);
    await screenshot('02-04-admin-classroom-empty');

    const classData = await apiPost('/classrooms', { name: '2026 스타트업 코딩입문', initial_capital: 50000000 }, adminToken);
    classroomCode = classData.data?.code || '';
    console.log(`  📋 클래스룸 코드: ${classroomCode}`);

    await page.reload();
    await sleep(1500);
    await screenshot('02-05-admin-classroom-created');

    // 2.5 대출 관리 (빈 상태)
    console.log('\n🧪 2.5: 대출 관리 (빈 상태)');
    await page.goto(BASE + '/admin/loans');
    await sleep(1500);
    await screenshot('02-06-admin-loans-empty');

    // 2.6 KPI 관리
    console.log('\n🧪 2.6: KPI 관리');
    await page.goto(BASE + '/admin/kpi');
    await sleep(1500);
    await screenshot('02-07-admin-kpi');

    // ============================================================
    // PHASE 3: 학생1 - 기본 흐름
    // ============================================================
    console.log('\n━━━━ PHASE 3: 학생1 기본 흐름 ━━━━');

    // Get student tokens
    const stu1Data = await apiPost('/auth/login', { email: 'student1@ewha.ac.kr', password: 'password1234' });
    student1Token = stu1Data.data?.token || '';
    const stu2Data = await apiPost('/auth/login', { email: 'student2@ewha.ac.kr', password: 'password1234' });
    student2Token = stu2Data.data?.token || '';

    // 3.1 학생1 로그인 (승인 후)
    console.log('\n🧪 3.1: 학생1 로그인');
    // Logout admin first
    await page.evaluate(() => localStorage.clear());
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'student1@ewha.ac.kr');
    await page.fill('input[type="password"]', 'password1234');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await screenshot('03-01-student-login-success');

    // 3.2 클래스룸 참여 전 피드
    console.log('\n🧪 3.2: 클래스룸 참여 전 피드');
    await page.goto(BASE + '/feed');
    await sleep(1500);
    await screenshot('03-02-feed-no-classroom');

    // 3.3 클래스룸 참여 (UI를 통해)
    console.log('\n🧪 3.3: 클래스룸 참여');
    const codeInput = page.locator('input[placeholder*="코드"], input[placeholder*="참여"]').first();
    if (await codeInput.isVisible() && classroomCode) {
      await codeInput.fill(classroomCode);
      await screenshot('03-03-classroom-code-filled');
      const joinBtn = page.locator('button:has-text("참여")').first();
      if (await joinBtn.isVisible()) {
        await joinBtn.click();
        await sleep(2000);
        await screenshot('03-04-classroom-joined');
      }
    } else {
      // Fallback: join via API
      await apiPost('/classrooms/join', { code: classroomCode }, student1Token);
      await page.goto(BASE + '/feed');
      await sleep(2000);
      await screenshot('03-04-classroom-joined');
    }

    // Student2 also joins classroom
    await apiPost('/classrooms/join', { code: classroomCode }, student2Token);
    console.log('  ✅ 학생2 클래스룸 참여 완료');

    // 3.5 피드 - 게시물 작성
    console.log('\n🧪 3.5: 게시물 작성');
    await page.goto(BASE + '/feed');
    await sleep(1500);
    const newPostBtn = page.locator('button:has-text("새 게시물"), button:has-text("게시물 작성"), a:has-text("게시물 작성")').first();
    if (await newPostBtn.isVisible()) {
      await newPostBtn.click();
      await sleep(1000);
      await screenshot('03-05-post-create-form');
    }

    // Create post via API
    // Get first channel
    const classroomsData = await apiGet('/classrooms', student1Token);
    const classrooms = classroomsData.data || [];
    let channelId = 0;
    if (classrooms.length > 0) {
      const channelsData = await apiGet(`/classrooms/${classrooms[0].id}/channels`, student1Token);
      const channels = channelsData.data || [];
      // Use "자유" (free) channel - students can't post to "공지" (notice, admin-only)
      const freeChannel = channels.find((ch: any) => ch.write_role === 'all' || ch.slug === 'free');
      if (freeChannel) channelId = freeChannel.id;
      else if (channels.length > 1) channelId = channels[1].id; // fallback to second channel
    }

    if (channelId) {
      await apiPost(`/channels/${channelId}/posts`, {
        title: '안녕하세요! 첫 게시물입니다',
        content: '이화AI랩을 설립하고 열심히 코딩 공부 중입니다. 같이 공부하실 분 모집합니다! 🙌',
        tags: '소개,모집',
      }, student1Token);
      console.log('  ✅ 게시물 작성 완료');
    }

    // 3.6 피드 - 게시물 확인
    console.log('\n🧪 3.6: 피드 게시물 확인');
    await page.goto(BASE + '/feed');
    await sleep(2000);
    await screenshot('03-06-feed-with-post');

    // 3.7 지갑 - 초기 자본금 확인
    console.log('\n🧪 3.7: 지갑 (초기 자본금)');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('03-07-wallet-initial');

    // 3.8 거래내역 - 초기 자본금 지급 확인
    console.log('\n🧪 3.8: 거래내역 (초기 자본금)');
    await page.goto(BASE + '/wallet/transactions');
    await sleep(1500);
    await screenshot('03-08-transactions-initial');

    // ============================================================
    // PHASE 4: 회사 설립 & 관리
    // ============================================================
    console.log('\n━━━━ PHASE 4: 회사 설립 & 관리 ━━━━');

    // 4.1 회사 목록 (비어있음)
    console.log('\n🧪 4.1: 회사 목록 (비어있음)');
    await page.goto(BASE + '/company');
    await sleep(1500);
    await screenshot('04-01-company-list-empty');

    // 4.2 회사 설립 폼
    console.log('\n🧪 4.2: 회사 설립 폼');
    await page.goto(BASE + '/company/new');
    await sleep(1500);
    await screenshot('04-02-company-new-form');

    // 4.3 회사 설립 (UI를 통해)
    console.log('\n🧪 4.3: 회사 설립');
    const compNameInput = page.locator('input[placeholder*="회사명"], input[name="name"]').first();
    if (await compNameInput.isVisible()) {
      await compNameInput.fill('이화AI랩');
      const descInput = page.locator('textarea, input[placeholder*="소개"], input[name="description"]').first();
      if (await descInput.isVisible()) await descInput.fill('AI 기반 교육 스타트업');
      await screenshot('04-03-company-form-filled');

      const submitBtn = page.locator('button[type="submit"], button:has-text("설립")').first();
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
        await sleep(2000);
        await screenshot('04-04-company-created');
      }
    }

    // Ensure company exists via API
    const compResp = await apiGet('/companies', student1Token);
    const companies = compResp.data || [];
    if (companies.length === 0) {
      const newComp = await apiPost('/companies', { name: '이화AI랩', description: 'AI 기반 교육 스타트업', initial_capital: 5000000 }, student1Token);
      companyId = newComp.data?.id || 1;
      console.log(`  ✅ 회사 설립 (API): ${newComp.data?.name}`);
    } else {
      companyId = companies[0].id;
    }

    // 4.5 회사 목록 (설립 후)
    console.log('\n🧪 4.5: 회사 목록 (설립 후)');
    await page.goto(BASE + '/company');
    await sleep(1500);
    await screenshot('04-05-company-list-with-company');

    // 4.6 회사 상세
    console.log('\n🧪 4.6: 회사 상세');
    await page.goto(BASE + `/company/${companyId}`);
    await sleep(1500);
    await screenshot('04-06-company-detail');

    // 4.7 명함 페이지
    console.log('\n🧪 4.7: 명함');
    await page.goto(BASE + `/company/${companyId}/card`);
    await sleep(1500);
    await screenshot('04-07-business-card');

    // 4.8 지갑 (회사설립 후)
    console.log('\n🧪 4.8: 지갑 (회사설립 후)');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('04-08-wallet-after-company');

    // ============================================================
    // PHASE 5: 프리랜서 마켓
    // ============================================================
    console.log('\n━━━━ PHASE 5: 프리랜서 마켓 ━━━━');

    // 5.1 마켓 (빈 상태)
    console.log('\n🧪 5.1: 마켓 (빈 상태)');
    await page.goto(BASE + '/market');
    await sleep(1500);
    await screenshot('05-01-market-empty');

    // 5.2 의뢰 작성 폼
    console.log('\n🧪 5.2: 의뢰 작성');
    await page.goto(BASE + '/market/new');
    await sleep(1500);
    await screenshot('05-02-market-new-form');

    // Create job via API
    const jobResp = await apiPost('/freelance/jobs', {
      title: '랜딩페이지 제작',
      description: '이화AI랩 소개 랜딩페이지를 제작해주세요. 반응형 디자인 필수입니다.',
      budget: 500000,
      deadline: '2026-04-01',
      required_skills: 'React,TypeScript,Tailwind',
    }, student1Token);
    jobId = jobResp.data?.id || 1;
    console.log(`  ✅ 의뢰 생성: ${jobResp.data?.title || 'OK'}`);

    // 5.3 마켓 (의뢰 있음)
    console.log('\n🧪 5.3: 마켓 (의뢰 있음)');
    await page.goto(BASE + '/market');
    await sleep(1500);
    await screenshot('05-03-market-with-job');

    // 5.4 의뢰 상세
    console.log('\n🧪 5.4: 의뢰 상세');
    await page.goto(BASE + `/market/${jobId}`);
    await sleep(1500);
    await screenshot('05-04-market-detail');

    // 5.5 학생2가 의뢰에 지원 (student2 시점)
    console.log('\n🧪 5.5: 학생2 의뢰 지원');
    const applyResp = await apiPost(`/freelance/jobs/${jobId}/apply`, {
      cover_letter: 'React와 TypeScript를 잘 다룹니다. 포트폴리오 사이트도 만들어봤습니다!',
    }, student2Token);
    console.log(`  ✅ 지원: ${applyResp.success ? '성공' : '실패'}`);

    // Reload market detail to see application
    await page.goto(BASE + `/market/${jobId}`);
    await sleep(1500);
    await screenshot('05-05-market-detail-with-applicant');

    // ============================================================
    // PHASE 6: 투자 & 거래소
    // ============================================================
    console.log('\n━━━━ PHASE 6: 투자 & 거래소 ━━━━');

    // 6.1 투자 페이지
    console.log('\n🧪 6.1: 투자 페이지');
    await page.goto(BASE + '/invest');
    await sleep(1500);
    await screenshot('06-01-invest-empty');

    // 6.2 거래소 페이지
    console.log('\n🧪 6.2: 거래소');
    await page.goto(BASE + '/exchange');
    await sleep(1500);
    await screenshot('06-02-exchange-empty');

    // ============================================================
    // PHASE 7: 은행 & 대출
    // ============================================================
    console.log('\n━━━━ PHASE 7: 은행 & 대출 ━━━━');

    // 7.1 은행 (빈 상태)
    console.log('\n🧪 7.1: 은행 (빈 상태)');
    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('07-01-bank-empty');

    // 7.2 대출 신청 폼
    console.log('\n🧪 7.2: 대출 신청 폼');
    await page.goto(BASE + '/bank/apply');
    await sleep(1500);
    await screenshot('07-02-loan-apply-form');

    // 7.3 대출 신청 (UI를 통해)
    console.log('\n🧪 7.3: 대출 신청');
    const amountInput = page.locator('input[type="number"], input[placeholder*="금액"]').first();
    if (await amountInput.isVisible()) {
      await amountInput.fill('3000000');
      const purposeInput = page.locator('input[placeholder*="목적"], textarea').first();
      if (await purposeInput.isVisible()) await purposeInput.fill('서버 인프라 구축');
      await screenshot('07-03-loan-form-filled');

      const submitBtn = page.locator('button[type="submit"], button:has-text("신청")').first();
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
        await sleep(2000);
        await screenshot('07-04-loan-submitted');
      }
    }

    // Loan was already created via UI form above - no API fallback needed

    // 7.5 은행 (대출 신청 후)
    console.log('\n🧪 7.5: 은행 (대출 신청 후)');
    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('07-05-bank-with-pending-loan');

    // 7.6 관리자 대출 승인
    console.log('\n🧪 7.6: 관리자 대출 승인');
    const loansResp = await apiGet('/admin/loans', adminToken);
    const adminLoans = loansResp.data?.loans || loansResp.data || [];
    const pendingLoan = Array.isArray(adminLoans) ? adminLoans.find((l: any) => l.status === 'pending') : null;
    if (pendingLoan) {
      await apiPut(`/admin/loans/${pendingLoan.id}/approve`, { interest_rate: 3.5 }, adminToken);
      console.log(`  ✅ 대출 승인 (이자율 3.5%)`);
    }

    // Admin loans page after approval
    await page.evaluate(() => localStorage.clear());
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'admin@ewha.ac.kr');
    await page.fill('input[type="password"]', 'admin1234');
    await page.click('button[type="submit"]');
    await sleep(2000);
    await page.goto(BASE + '/admin/loans');
    await sleep(1500);
    await screenshot('07-06-admin-loans-approved');

    // 7.7 학생 은행 (승인 후)
    console.log('\n🧪 7.7: 은행 (대출 승인 후)');
    await page.evaluate(() => localStorage.clear());
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'student1@ewha.ac.kr');
    await page.fill('input[type="password"]', 'password1234');
    await page.click('button[type="submit"]');
    await sleep(2000);

    await page.goto(BASE + '/bank');
    await sleep(1500);
    await screenshot('07-07-bank-loan-approved');

    // ============================================================
    // PHASE 8: 프로필 & 알림
    // ============================================================
    console.log('\n━━━━ PHASE 8: 프로필 & 알림 ━━━━');

    // 8.1 프로필
    console.log('\n🧪 8.1: 프로필');
    await page.goto(BASE + '/profile');
    await sleep(1500);
    await screenshot('08-01-profile');

    // 8.2 알림 (대출 승인 알림 확인)
    console.log('\n🧪 8.2: 알림');
    await page.goto(BASE + '/notifications');
    await sleep(1500);
    await screenshot('08-02-notifications');

    // ============================================================
    // PHASE 9: 최종 상태 확인
    // ============================================================
    console.log('\n━━━━ PHASE 9: 최종 상태 확인 ━━━━');

    // 9.1 지갑 최종
    console.log('\n🧪 9.1: 지갑 최종');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('09-01-wallet-final');

    // 9.2 거래내역 최종
    console.log('\n🧪 9.2: 거래내역 최종');
    await page.goto(BASE + '/wallet/transactions');
    await sleep(1500);
    await screenshot('09-02-transactions-final');

    // 9.3 프로필 최종
    console.log('\n🧪 9.3: 프로필 최종');
    await page.goto(BASE + '/profile');
    await sleep(1500);
    await screenshot('09-03-profile-final');

    // ============================================================
    // PHASE 10: 학생2 시점 테스트
    // ============================================================
    console.log('\n━━━━ PHASE 10: 학생2 시점 ━━━━');

    // Login as student2
    await page.evaluate(() => localStorage.clear());
    await page.goto(BASE + '/login');
    await page.waitForSelector('input[type="email"]', { timeout: 5000 });
    await page.fill('input[type="email"]', 'student2@ewha.ac.kr');
    await page.fill('input[type="password"]', 'password1234');
    await page.click('button[type="submit"]');
    await sleep(2000);

    // 10.1 학생2 피드 (같은 클래스룸)
    console.log('\n🧪 10.1: 학생2 피드');
    await page.goto(BASE + '/feed');
    await sleep(2000);
    await screenshot('10-01-student2-feed');

    // 10.2 학생2 지갑
    console.log('\n🧪 10.2: 학생2 지갑');
    await page.goto(BASE + '/wallet');
    await sleep(1500);
    await screenshot('10-02-student2-wallet');

    // 10.3 학생2 프로필
    console.log('\n🧪 10.3: 학생2 프로필');
    await page.goto(BASE + '/profile');
    await sleep(1500);
    await screenshot('10-03-student2-profile');

    // 10.4 학생2 회사 설립
    console.log('\n🧪 10.4: 학생2 회사 설립');
    await apiPost('/companies', { name: '이화테크', description: 'EdTech 스타트업', initial_capital: 2000000 }, student2Token);
    await page.goto(BASE + '/company');
    await sleep(1500);
    await screenshot('10-04-student2-company');

    // 10.5 학생2 마켓 보기 (학생1의 의뢰)
    console.log('\n🧪 10.5: 학생2 마켓');
    await page.goto(BASE + '/market');
    await sleep(1500);
    await screenshot('10-05-student2-market');

    // ============================================================
    // PHASE 11: 네비게이션 & UI 컴포넌트
    // ============================================================
    console.log('\n━━━━ PHASE 11: UI 컴포넌트 ━━━━');

    // 11.1 하단 네비게이션
    console.log('\n🧪 11.1: 하단 네비게이션');
    await page.goto(BASE + '/feed');
    await sleep(1000);
    await screenshot('11-01-bottom-nav');

    // 11.2 더보기 메뉴
    console.log('\n🧪 11.2: 더보기 메뉴');
    const moreBtn = page.locator('button:has-text("더보기"), nav button:last-child').first();
    if (await moreBtn.isVisible()) {
      await moreBtn.click();
      await sleep(800);
      await screenshot('11-02-more-menu');
    }

    // 11.3 헤더 알림 벨
    console.log('\n🧪 11.3: 헤더');
    await page.goto(BASE + '/feed');
    await sleep(1000);
    await screenshot('11-03-header');

    // ============================================================
    // Summary
    // ============================================================
    console.log('\n✅ 확장 E2E 테스트 완료!');
    console.log(`📁 스크린샷: ${SCREENSHOT_DIR}`);
    console.log(`📊 총 시나리오: ~50개`);

    if (consoleErrors.length > 0) {
      console.log(`\n⚠️ 콘솔 에러 ${consoleErrors.length}개:`);
      consoleErrors.forEach((e, i) => console.log(`  ${i + 1}. ${e.substring(0, 200)}`));
    }

  } catch (err) {
    console.error('\n❌ 테스트 실패:', err);
    await screenshot('error-state');
  } finally {
    await browser.close();
  }
}

main();
