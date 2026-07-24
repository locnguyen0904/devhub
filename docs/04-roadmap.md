# 04 — Roadmap

Ước lượng theo **tuần làm việc bán thời gian** (~15 giờ/tuần) cho một người. Mỗi phase có tiêu chí hoàn thành cụ thể — không có "xong khoảng 80%".

Nguyên tắc xuyên suốt: mỗi phase kết thúc bằng một thứ **chạy được và demo được**, không phải một tầng kiến trúc chưa nối vào đâu.

---

## Phase 0 — Nền móng (1 tuần)

Dựng bộ khung để mọi phase sau chỉ việc thêm module.

- `docker-compose.yml`: Postgres 16, Redis 7, MinIO
- Backend: `cmd/api`, load config từ env (fail-fast nếu thiếu biến), `chi` router, chuỗi middleware, graceful shutdown
- `platform/`: pgxpool, redis client, slog, `httpx` (JSON + map lỗi)
- `golang-migrate` + `sqlc`, chạy migration `000001_init` với schema §3 và §4 của [02-data-model.md](02-data-model.md)
- Frontend: Vite + React + TS, Tailwind + shadcn, React Router, TanStack Query provider, layout khung
- `Makefile`: `dev`, `migrate-up/down`, `sqlc`, `lint`, `test`
- CI (GitHub Actions): `go vet`, `golangci-lint`, `go test`, `tsc --noEmit`, `eslint`

**Xong khi:** `make dev && make api` lên được, `GET /healthz` trả 200, `GET /readyz` báo đúng trạng thái Postgres + Redis, `pnpm dev` render được trang trống có header. CI xanh.

---

## Phase 1 — Xác thực (1 tuần)

Toàn bộ luồng ở [03-api.md §2](03-api.md#2-luồng-xác-thực).

- OAuth app trên GitHub, cấu hình callback URL
- `modules/auth`: redirect, callback, kiểm tra `state` qua Redis, đổi code lấy token, gọi GitHub API lấy hồ sơ
- Tìm-hoặc-tạo `users` + `oauth_accounts` trong một transaction
- Cấp JWT, xoay vòng refresh token, phát hiện tái sử dụng token đã thu hồi
- Middleware `Auth` (bắt buộc) và `OptionalAuth` (gắn user nếu có token)
- Frontend: nút đăng nhập, `AuthProvider`, fetch client tự refresh khi gặp 401 (gộp các request đang chờ, chỉ refresh một lần)
- Mã hoá `access_token_enc` bằng AES-GCM

**Xong khi:** đăng nhập bằng tài khoản GitHub thật, F5 vẫn giữ phiên, đăng xuất thu hồi được token, dùng lại refresh token cũ bị chặn và huỷ cả family.

**Chỗ dễ sai:** không gộp refresh thì mở 5 tab sẽ bắn 5 request refresh song song, 4 cái sau dùng token vừa bị xoay vòng → người dùng bị đá ra ngoài.

---

## Phase 2 — Bài viết & Editor (2 tuần)

Phần lõi. Dài nhất, và đáng dành thời gian nhất.

**Backend**
- `modules/post`: CRUD, publish/unpublish, sinh slug (bỏ dấu tiếng Việt), tính thời gian đọc
- `platform/markdown`: `goldmark` (GFM, bảng, tô màu code qua `chroma`, tự sinh id cho heading) → `bluemonday` sanitize
- `modules/tag`: tự tạo tag mới, gắn tag, autocomplete
- `/uploads/presign` cho ảnh
- Phân quyền: chỉ tác giả sửa/xoá được bài của mình

**Frontend**
- Editor: CodeMirror 6 chế độ markdown, xem trước cạnh bên, tự lưu nháp mỗi 3 giây (debounce)
- Trang bài viết: render HTML, mục lục, nút copy khối code
- Trang quản lý bài của tôi: nháp / đã xuất bản

**Xong khi:** viết bài có code block và ảnh, lưu nháp, xuất bản, mở link ẩn danh ở tab riêng và đọc được.

**Chỗ dễ sai:** sanitize HTML là bắt buộc, không phải tuỳ chọn. Markdown cho phép nhúng HTML thô — bỏ qua bước này là mở cửa cho XSS lưu trữ, kẻ tấn công đăng một bài là chiếm được phiên của mọi người đọc.

---

## Phase 3 — Feed, Tag, Tìm kiếm (1 tuần)

- `modules/feed`: sắp xếp `latest` và `hot` ([02-data-model.md §8](02-data-model.md#8-xếp-hạng-feed-nổi-bật)), phân trang cursor
- Cache feed `hot` trong Redis, TTL 5 phút
- Trang tag: bài theo tag
- `/search` dùng `tsvector`, có làm nổi bật từ khoá (`ts_headline`)
- Frontend: cuộn vô hạn (`useInfiniteQuery`), skeleton loading
- Gộp lượt xem qua Redis + goroutine flush mỗi 60 giây

**Xong khi:** feed cuộn vô hạn mượt không lặp bài, lọc tag đúng, tìm kiếm trả kết quả hợp lý dưới 200 ms với 10k bài giả lập.

**Cần kiểm chứng:** seed 10k bài rồi `EXPLAIN ANALYZE` truy vấn feed. Nếu thấy `Seq Scan` thì partial index chưa được dùng — sửa ngay ở đây, đừng đợi tới lúc có dữ liệu thật.

---

## Phase 4 — Bình luận & Reaction (1 tuần)

- `modules/comment`: cây 2 tầng, một query lấy hết rồi gom nhóm trong Go, xoá mềm giữ nguyên cây
- `modules/reaction`: `PUT`/`DELETE` idempotent, cập nhật bộ đếm trong cùng transaction
- Bookmark + trang "Đã lưu"
- Frontend: luồng trả lời, cập nhật lạc quan cho reaction (rollback nếu request hỏng)
- Rate limit cho endpoint ghi

**Xong khi:** bình luận, trả lời, sửa trong 30 phút đầu, xoá; bấm reaction 10 lần liên tục vẫn ra đúng 1; số đếm khớp `COUNT(*)` sau khi kiểm tra chéo.

---

## Phase 5 — Hoàn thiện để phát hành (1 tuần)

- SEO: thẻ meta, Open Graph, JSON-LD `Article`, `sitemap.xml`, `robots.txt`
- Prerender trang bài viết cho bot mạng xã hội (Cloudflare Worker hoặc `vite-plugin-ssr`)
- Ảnh OG sinh tự động từ tiêu đề
- Metrics Prometheus, dashboard cơ bản
- Trang lỗi 404/500, error boundary
- Kiểm tra khả năng tiếp cận: điều hướng bàn phím, độ tương phản, nhãn ARIA
- Dockerfile multi-stage, deploy lên một VPS, HTTPS qua Caddy
- Sao lưu Postgres tự động hằng ngày

**Xong khi:** dán link bài viết vào Slack/Twitter hiện đúng ảnh preview; Lighthouse ≥ 90 cả 4 mục; sao lưu chạy được và **đã thử phục hồi ít nhất một lần**.

---

## Tổng MVP: ~7 tuần

| Phase | Nội dung | Tuần |
|---|---|---|
| 0 | Nền móng | 1 |
| 1 | Xác thực | 1 |
| 2 | Bài viết & Editor | 2 |
| 3 | Feed, Tag, Tìm kiếm | 1 |
| 4 | Bình luận & Reaction | 1 |
| 5 | Hoàn thiện | 1 |

---

## Sau MVP

Thứ tự đề xuất, dựa trên tỉ lệ giá trị/công sức:

**Phase 6 — Social graph (1 tuần).** Follow, feed cá nhân hoá, thông báo. Rẻ và biến trang web từ "nơi đăng bài" thành "nơi quay lại".

**Phase 7 — Profile + GitHub sync (2 tuần).** Đồng bộ repo/contribution qua GitHub API, showcase project, tech stack. Đây là thứ khiến DevHub khác Dev.to. Cần worker nền + tôn trọng rate limit của GitHub + cache mạnh (dữ liệu GitHub đổi chậm, cache 6 giờ là đủ).

**Phase 8 — Notes / Workspace (4+ tuần).** Editor dạng block kiểu Notion. Đắt hơn hai phase kia cộng lại — kéo thả, lồng nhau, slash command, đồng bộ realtime là cả một sản phẩm riêng. Chỉ làm khi hai trục kia đã có người dùng thật. Cân nhắc dùng sẵn TipTap/BlockNote thay vì tự viết.

**Phase 9 — Series bài viết, newsletter, RSS.** Nhỏ lẻ, làm xen kẽ khi cần đổi không khí.

---

## Rủi ro cần theo dõi

| Rủi ro | Dấu hiệu sớm | Đối phó |
|---|---|---|
| Phase 2 kéo dài vô hạn | Editor "gần xong" sang tuần thứ ba | Chốt cứng: markdown + xem trước. WYSIWYG là Phase 8 |
| Spam khi mở công khai | Bài chứa link lạ xuất hiện | Đã có rate limit; thêm ngưỡng tài khoản mới + báo cáo vi phạm |
| Full-text search chậm dần | p95 `/search` vượt 500 ms | Đổi sang Meilisearch, API không đổi |
| Kéo thêm tính năng vào MVP | "Thêm cái này nhanh thôi" | Phase 6–9 tồn tại đúng để chứa những ý tưởng đó |
