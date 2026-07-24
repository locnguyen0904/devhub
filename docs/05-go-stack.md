# 05 — Go Tech Stack

Toolchain: **Go 1.24** (đã có sẵn trên máy). Version trong tài liệu này được truy vấn từ Go module proxy ngày 2026-07-24 — chốt version cụ thể trong `go.mod`, không dùng `latest`.

Nguyên tắc chọn thư viện:

1. **Thư viện chuẩn trước.** `net/http`, `log/slog`, `database/sql`, `context`, `errors` đã đủ tốt. Chỉ thêm dependency khi nó giải quyết một vấn đề thật.
2. **Ưu tiên thư viện, tránh framework.** Thư viện thì gỡ ra được; framework thì viết lại cả dự án.
3. **Không dùng thư viện đã ngừng bảo trì**, kể cả khi nó vẫn chạy tốt — lỗ hổng bảo mật sẽ không có ai vá.

---

## 1. Bảng tổng hợp

| Vai trò | Thư viện | Version |
|---|---|---|
| Router HTTP | `github.com/go-chi/chi/v5` | v5.3.1 |
| Driver Postgres | `github.com/jackc/pgx/v5` | v5.10.0 |
| Sinh code truy vấn | `github.com/sqlc-dev/sqlc` (CLI) | v1.31.1 |
| Migration | `github.com/golang-migrate/migrate/v4` | v4.19.1 |
| Redis | `github.com/redis/go-redis/v9` | v9.21.0 |
| Rate limit | `github.com/go-redis/redis_rate/v10` | v10.0.1 |
| OAuth 2.0 | `golang.org/x/oauth2` | v0.36.0 |
| JWT | `github.com/golang-jwt/jwt/v5` | v5.3.1 |
| UUID v7 | `github.com/google/uuid` | v1.6.0 |
| Markdown → HTML | `github.com/yuin/goldmark` | v1.8.4 |
| Tô màu code | `github.com/alecthomas/chroma/v2` | v2.27.0 |
| Sanitize HTML | `github.com/microcosm-cc/bluemonday` | v1.0.27 |
| Sinh slug | `github.com/gosimple/slug` | v1.15.0 |
| Validate input | `github.com/go-playground/validator/v10` | v10.30.3 |
| Đọc config | `github.com/caarlos0/env/v11` | v11.4.1 |
| S3 / MinIO | `github.com/aws/aws-sdk-go-v2/service/s3` | v1.106.0 |
| Metrics | `github.com/prometheus/client_golang` | v1.24.0 |
| Assertion test | `github.com/stretchr/testify` | v1.11.1 |
| DB thật khi test | `github.com/testcontainers/testcontainers-go` | v0.43.0 |
| Logging | `log/slog` — **thư viện chuẩn** | — |
| Lint | `golangci-lint` (CLI) | v2.12.2 |
| Hot reload | `air` (CLI) | v1.67.2 |

Tổng: **17 dependency trực tiếp**. Con số này quan trọng — mỗi dependency là một thứ phải theo dõi CVE và nâng cấp.

---

## 2. Lý do từng lựa chọn quan trọng

### Router: `chi` — không phải Gin, không phải Echo

`chi` là router thuần trên `net/http`. Handler có chữ ký `func(http.ResponseWriter, *http.Request)` chuẩn, middleware là `func(http.Handler) http.Handler` chuẩn.

Hệ quả thực tế: mọi middleware của hệ sinh thái Go dùng được ngay, và nếu sau này bỏ `chi` thì handler không phải viết lại dòng nào. Gin dùng `*gin.Context` riêng — chọn Gin là khoá toàn bộ handler vào Gin vĩnh viễn.

**Vì sao không dùng `net/http` trần** (Go 1.22+ đã hỗ trợ `GET /posts/{id}`): `chi` cho `r.Route()` lồng nhóm và `r.Group()` để gắn middleware cho một nhóm route. Với ~40 endpoint chia theo module thì thứ này tiết kiệm thật; kích thước thư viện gần như bằng không.

### Truy cập DB: `pgx` + `sqlc` — không phải GORM

**`pgx` bỏ qua `database/sql`**, nói trực tiếp protocol của Postgres. Được: prepared statement cache, `COPY` protocol, hỗ trợ đúng kiểu riêng của Postgres (`jsonb`, `tsvector`, `inet`, mảng), và `pgxpool` là connection pool tốt nhất trong hệ sinh thái Go.

**`sqlc` sinh code Go từ file SQL viết tay.** Luồng làm việc:

```sql
-- db/queries/post.sql
-- name: ListPublishedPosts :many
SELECT p.*, u.username, u.avatar_url
FROM posts p JOIN users u ON u.id = p.author_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND (p.published_at, p.id) < (@cursor_at, @cursor_id)
ORDER BY p.published_at DESC, p.id DESC
LIMIT @lim;
```

`make sqlc` sinh ra hàm Go có kiểu đầy đủ. `sqlc` **kết nối vào schema thật để kiểm tra SQL lúc sinh code** — gõ sai tên cột là lỗi lúc build, không phải lỗi lúc chạy ở production.

Vì sao không GORM:

| | `sqlc` | GORM |
|---|---|---|
| SQL sinh ra | chính là SQL bạn viết | phải đọc log mới biết |
| Vấn đề N+1 | không thể xảy ra | mặc định dễ dính |
| Cursor pagination `(a,b) < (c,d)` | viết thẳng | phải `Raw()` |
| Query `hot` với công thức xếp hạng | viết thẳng | phải `Raw()` |
| `INSERT ... ON CONFLICT DO NOTHING` | viết thẳng | API riêng, khác biệt tinh vi |
| Lỗi sai tên cột | lúc build | lúc chạy |

Data model ở [02-data-model.md](02-data-model.md) có partial index, tsvector, tuple comparison, công thức xếp hạng — nghĩa là phần lớn query nóng sẽ phải viết SQL thô kiểu gì cũng vậy. Vậy thì mang thêm một ORM vào để làm gì.

Cái giá của `sqlc`: cập nhật một phần (`PATCH /posts/{id}` chỉ sửa vài trường) hơi vướng. Cách xử lý — dùng `COALESCE`:

```sql
-- name: UpdatePost :one
UPDATE posts SET
    title         = COALESCE(sqlc.narg('title'), title),
    body_markdown = COALESCE(sqlc.narg('body_markdown'), body_markdown),
    updated_at    = now()
WHERE id = @id AND author_id = @author_id
RETURNING *;
```

### Migration: `golang-migrate` — không phải `goose`

Cả hai đều tốt. Chọn `golang-migrate` vì nó tách bạch `.up.sql`/`.down.sql` thành hai file riêng, và hỗ trợ `CREATE INDEX CONCURRENTLY` (chạy ngoài transaction) — thứ này bắt buộc phải có khi thêm index vào bảng đã có dữ liệu ở production.

Chạy migration bằng **CLI như một bước deploy riêng**, không nhúng vào binary API. Nhúng vào nghĩa là chạy 3 instance thì cả 3 cùng tranh nhau migrate.

### Logging: `log/slog` — không phải `zerolog`/`zap`

Thư viện chuẩn từ Go 1.21. Có structured logging, JSON handler, `slog.Group`, và quan trọng nhất là `Handler` interface để gắn thêm hành vi. Đủ nhanh cho web API (bottleneck là DB và network, không phải logger).

Một handler tự viết bơm `request_id` và `user_id` từ context vào mọi dòng log, không cần truyền logger xuyên qua mọi lời gọi hàm:

```go
func (h *ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	if uid, ok := ctx.Value(ctxKeyUserID).(uuid.UUID); ok {
		r.AddAttrs(slog.String("user_id", uid.String()))
	}
	return h.next.Handle(ctx, r)
}
```

### Markdown: `goldmark` + `chroma` + `bluemonday`

`goldmark` là engine đứng sau Hugo — đúng chuẩn CommonMark, kiến trúc extension sạch. Bật: GFM (bảng, strikethrough, tự nhận link), footnote, tự sinh `id` cho heading (để làm mục lục), và `chroma` để tô màu code.

**`bluemonday` không phải tuỳ chọn.** Markdown cho phép nhúng HTML thô; không sanitize là mở cửa cho stored XSS — đăng một bài là chiếm được phiên của mọi người đọc. Thứ tự bắt buộc:

```go
var policy = bluemonday.UGCPolicy()   // cấu hình một lần, dùng lại

func Render(md string) (string, error) {
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return policy.Sanitize(buf.String()), nil   // sanitize SAU khi render
}
```

Sanitize markdown *trước* khi render là sai — nó không phải HTML nên policy không hiểu, và ký tự markdown hợp lệ sẽ bị phá.

Policy cần nới thêm cho `<span class="...">` mà `chroma` sinh ra, và `class` trên `<code>`:

```go
policy.AllowAttrs("class").Matching(regexp.MustCompile(`^(chroma|language-|hl)`)).
    OnElements("span", "code", "pre", "div")
```

### UUID: `google/uuid` v1.6.0

Từ v1.6.0 có `uuid.NewV7()` — UUID sắp theo thời gian. Quan trọng vì khoá chính ngẫu nhiên (v4) khiến B-tree index của Postgres phân mảnh: mỗi lần chèn rơi vào một trang ngẫu nhiên. UUIDv7 chèn tuần tự ở cuối index, giống `bigserial`, nhưng không lộ số lượng bản ghi ra URL.

### Rate limit: `redis_rate` — không tự viết

`redis_rate` cài đặt GCRA (generic cell rate algorithm) bằng Lua script chạy nguyên tử trên Redis. Tự viết sliding window bằng `INCR` + `EXPIRE` có race condition kinh điển: `INCR` xong mà process chết trước `EXPIRE` thì key sống mãi, khoá người dùng vĩnh viễn. Thư viện này chỉ có một dependency và làm đúng một việc.

### Config: `caarlos0/env` — không phải Viper

Config đọc từ biến môi trường (12-factor). Viper mang theo ~40 dependency gián tiếp để hỗ trợ YAML/TOML/etcd/Consul/hot-reload — dự án này không dùng thứ nào trong đó.

```go
type Config struct {
	Port         int           `env:"PORT" envDefault:"8080"`
	DatabaseURL  string        `env:"DATABASE_URL,required"`
	RedisURL     string        `env:"REDIS_URL,required"`
	JWTSecret    string        `env:"JWT_SECRET,required"`
	AccessTTL    time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	GitHubID     string        `env:"GITHUB_CLIENT_ID,required"`
	GitHubSecret string        `env:"GITHUB_CLIENT_SECRET,required"`
	AllowOrigins []string      `env:"CORS_ALLOW_ORIGINS" envSeparator:","`
}
```

`required` khiến thiếu biến là **sập lúc khởi động**, không phải `nil pointer` lúc 2 giờ sáng.

### Test: `testify` + `testcontainers-go`

- **Unit test service**: mock repository bằng interface tự viết tay. Repository interface trong mỗi module chỉ 5–8 method, mock tay ~30 dòng và đọc hiểu ngay. Chưa cần `mockery` ở quy mô này.
- **Integration test repository**: `testcontainers-go` bật một Postgres thật trong Docker, chạy migration, test SQL thật. Sqlite in-memory không thay thế được — nó không có `tsvector`, không có partial index, không có `ON CONFLICT` cùng cú pháp. Test trên thứ khác production là test một hệ thống khác.

Chạy container một lần cho cả package qua `TestMain`, mỗi test cuộn trong một transaction rồi rollback → sạch dữ liệu mà không phải bật lại container.

---

## 3. Sinh OpenAPI: hai hướng, cần bạn chọn

[03-api.md §10](03-api.md#10-sinh-mã-tự-động) yêu cầu sinh type TypeScript cho frontend từ spec. Có hai cách, khác nhau đáng kể:

**A. `chi` + `swaggo/swag` (v1.16.6)** — viết comment annotation trên mỗi handler, chạy `swag init` sinh spec.
Được: giữ nguyên `chi`, không đổi gì về kiến trúc.
Mất: comment và code có thể trôi lệch nhau mà vẫn build thành công. Spec sai âm thầm.

**B. `huma/v2` (v2.39.0)** — đăng ký operation bằng struct Go có tag; huma sinh OpenAPI 3.1 **từ chính kiểu dữ liệu**, đồng thời tự validate request theo schema đó. Chạy được trên `chi` làm router bên dưới.
Được: spec không thể lệch với code, vì nó *là* code. Bớt luôn `validator/v10`.
Mất: handler viết theo chữ ký của huma (`func(ctx, *Input) (*Output, error)`) — một lớp ràng buộc nữa, và ngược với nguyên tắc "tránh framework" ở trên.

Tôi nghiêng về **B** cho dự án này: bạn làm cả backend lẫn frontend một mình, nên spec trôi lệch là rủi ro thật và sẽ tốn nhiều giờ debug hơn là cái giá của việc bị buộc vào chữ ký handler. Nhưng đây là quyết định của bạn — nó chạm vào mọi handler nên đổi ý ở Phase 3 sẽ đắt.

---

## 4. Chưa dùng bây giờ, sẽ cần sau

| Thư viện | Version | Khi nào |
|---|---|---|
| `github.com/riverqueue/river` | v0.40.0 | Phase 7 — job queue chạy trên Postgres, thay bảng outbox tự viết khi GitHub sync cần retry/backoff/lập lịch |
| `github.com/meilisearch/meilisearch-go` | — | Khi p95 `/search` vượt 500 ms |
| `go.opentelemetry.io/otel` | — | Khi có nhiều hơn một service để trace xuyên qua |
| `github.com/vektra/mockery/v2` | v2.53.6 | Khi mock tay vượt quá ~10 interface |

---

## 5. `go.mod` dự kiến

```go
module github.com/locnguyen/devhub

go 1.24

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.106.0
	github.com/caarlos0/env/v11 v11.4.1
	github.com/go-chi/chi/v5 v5.3.1
	github.com/go-playground/validator/v10 v10.30.3
	github.com/go-redis/redis_rate/v10 v10.0.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/gosimple/slug v1.15.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/prometheus/client_golang v1.24.0
	github.com/redis/go-redis/v9 v9.21.0
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.43.0
	github.com/yuin/goldmark v1.8.4
	golang.org/x/oauth2 v0.36.0
)
```

Công cụ CLI (`sqlc`, `golangci-lint`, `air`, `migrate`) cài qua `go tool` directive của Go 1.24 — version bị ghim trong `go.mod`, cả máy dev lẫn CI dùng đúng một version, không phải cài toàn cục.

---

## 6. Một điểm cần kiểm chứng ở Phase 2

`gosimple/slug` chuyển chữ có dấu về ASCII qua transliteration. Cần một test khẳng định nó xử lý tiếng Việt đúng ý:

```go
func TestSlugVietnamese(t *testing.T) {
	require.Equal(t, "hieu-ve-goroutine-leak", slug.Make("Hiểu về goroutine leak"))
	require.Equal(t, "toi-uu-truy-van-postgresql", slug.Make("Tối ưu truy vấn PostgreSQL"))
}
```

Nếu nó xử lý `đ`/`Đ` không như mong muốn, thay bằng một hàm ~20 dòng dùng `golang.org/x/text/unicode/norm` (chuẩn hoá NFD rồi bỏ dấu thanh) — rẻ hơn nhiều so với việc phát hiện ra URL xấu sau khi đã có 200 bài viết.
