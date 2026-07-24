# DevHub

Nền tảng dành cho developer: viết bài kỹ thuật (Dev.to), hồ sơ nghề nghiệp đồng bộ GitHub, và không gian ghi chú dạng block (Notion).

**Trạng thái:** giai đoạn thiết kế. Chưa có code.

## Phạm vi giai đoạn 1 (MVP)

Chỉ làm **trục Blog/Posts** cho tới khi nó thực sự dùng được. Profile+GitHub sync và Notes/Workspace nằm ở giai đoạn sau, nhưng kiến trúc và data model bên dưới đã chừa chỗ sẵn cho chúng.

MVP bao gồm:

- Đăng nhập bằng GitHub OAuth
- Viết / sửa / xoá bài dạng Markdown, có nháp và xuất bản
- Trang bài viết công khai (SEO-friendly), trang tác giả
- Feed theo mới nhất / nổi bật, lọc theo tag
- Bình luận (2 tầng) và reaction
- Tìm kiếm toàn văn

## Stack

| Lớp | Lựa chọn |
|---|---|
| Backend | Go 1.24, `chi` router, modular monolith |
| Database | PostgreSQL 16, `pgx/v5` + `sqlc`, `golang-migrate` |
| Cache / rate limit | Redis 7 |
| Object storage | S3-compatible (MinIO khi dev) |
| Frontend | React 19 + TypeScript, Vite, TanStack Query, React Router, Tailwind + shadcn/ui |
| Auth | GitHub OAuth 2.0 → JWT access token + refresh token |
| Local dev | Docker Compose |

## Tài liệu

| Tài liệu | Nội dung |
|---|---|
| [docs/01-architecture.md](docs/01-architecture.md) | Kiến trúc modular monolith, cấu trúc thư mục, quy tắc ranh giới module |
| [docs/02-data-model.md](docs/02-data-model.md) | Schema PostgreSQL đầy đủ, index, chiến lược migration |
| [docs/03-api.md](docs/03-api.md) | REST API contract, luồng auth, phân trang, định dạng lỗi |
| [docs/04-roadmap.md](docs/04-roadmap.md) | Roadmap theo phase, tiêu chí hoàn thành từng phase |
| [docs/05-go-stack.md](docs/05-go-stack.md) | Thư viện Go: chọn gì, version nào, vì sao không chọn cái kia |

## Bắt đầu (sau khi có code)

```bash
make dev        # docker compose up: postgres, redis, minio
make migrate-up # chạy migration
make api        # chạy backend tại :8080
cd frontend && pnpm dev   # frontend tại :5173
```
