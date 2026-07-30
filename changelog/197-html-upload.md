# 게시글·댓글·DM HTML 파일 첨부

게시글·댓글의 공통 편집기와 DM에서 `.html`·`.htm` 파일을 첨부할 수 있게 했습니다. 같은 출처에서 HTML이 실행되지 않도록 다운로드 응답에는 `attachment`, `application/octet-stream`, `nosniff`를 적용했습니다.

프롬프트: “언러닝 게시글 및 댓글, dm 등 글 작성시 파일업로드 가능한곳에 html 업로드 가능하도록 해줘.”
